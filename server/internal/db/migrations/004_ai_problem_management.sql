ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS ai_insights JSONB,
ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES tickets(id),
ADD COLUMN IF NOT EXISTS is_problem BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_tickets_parent_id ON tickets(parent_id);
CREATE INDEX IF NOT EXISTS idx_tickets_is_problem ON tickets(is_problem);

CREATE OR REPLACE PROCEDURE sp_resolve_ticket(
    p_ticket_id UUID,
    p_actor_id UUID
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_ticket_status_code VARCHAR;
    v_due_at TIMESTAMPTZ;
    v_is_problem BOOLEAN;
    v_resolved_status_id SMALLINT;
BEGIN
    SELECT ts.name, t.due_at, t.is_problem
    INTO v_ticket_status_code, v_due_at, v_is_problem
    FROM tickets t
    JOIN ticket_statuses ts ON ts.id = t.status_id
    WHERE t.id = p_ticket_id;

    IF v_ticket_status_code NOT IN ('open', 'in_progress') THEN
        RAISE EXCEPTION 'Cannot resolve a % ticket', v_ticket_status_code;
    END IF;

    SELECT id INTO v_resolved_status_id
    FROM ticket_statuses
    WHERE name = 'resolved';

    UPDATE tickets
    SET status_id = v_resolved_status_id
    WHERE id = p_ticket_id;

    IF v_due_at IS NOT NULL AND NOW() > v_due_at THEN
        UPDATE tickets
        SET sla_breached = true
        WHERE id = p_ticket_id;
    END IF;

    IF COALESCE(v_is_problem, false) THEN
        UPDATE tickets
        SET status_id = v_resolved_status_id
        WHERE parent_id = p_ticket_id
          AND status_id IN (
              SELECT id
              FROM ticket_statuses
              WHERE name IN ('open', 'in_progress')
          );
    END IF;

    INSERT INTO audit_logs (id, ticket_id, actor_id, action, new_value)
    VALUES (
        gen_random_uuid(),
        p_ticket_id,
        p_actor_id,
        'resolved',
        jsonb_build_object('resolved_at', NOW())
    );
END;
$$;
