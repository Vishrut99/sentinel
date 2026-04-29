ALTER TABLE agents
ADD COLUMN IF NOT EXISTS skills JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS required_skills JSONB NOT NULL DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS assignment_justification TEXT,
ADD COLUMN IF NOT EXISTS skills_evaluated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_agents_skills ON agents USING GIN (skills);
CREATE INDEX IF NOT EXISTS idx_tickets_required_skills ON tickets USING GIN (required_skills);

DROP PROCEDURE IF EXISTS sp_assign_ticket(UUID, UUID, UUID);
DROP PROCEDURE IF EXISTS sp_assign_ticket(UUID, UUID, UUID, TEXT);

CREATE OR REPLACE PROCEDURE sp_assign_ticket(
	p_ticket_id UUID,
	p_agent_id UUID,
	p_actor_id UUID,
	p_justification TEXT DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
DECLARE
	v_agent_tickets INT;
	v_max_tickets INT;
	v_is_available BOOLEAN;
	v_ticket_status_code VARCHAR;
BEGIN
	SELECT ts.name INTO v_ticket_status_code
	FROM tickets t
	JOIN ticket_statuses ts ON ts.id = t.status_id
	WHERE t.id = p_ticket_id
	FOR UPDATE;

	IF v_ticket_status_code IN ('closed', 'resolved', 'cancelled') THEN
		RAISE EXCEPTION 'Cannot assign a % ticket', v_ticket_status_code;
	END IF;

	SELECT is_available, max_tickets INTO v_is_available, v_max_tickets
	FROM agents
	WHERE id = p_agent_id;

	IF NOT v_is_available THEN
		RAISE EXCEPTION 'Agent is not available';
	END IF;

	SELECT COUNT(*) INTO v_agent_tickets
	FROM tickets t
	JOIN ticket_statuses ts ON ts.id = t.status_id
	WHERE t.assigned_to = p_agent_id
	AND ts.name NOT IN ('resolved', 'closed', 'cancelled');

	IF v_agent_tickets >= v_max_tickets THEN
		RAISE EXCEPTION 'Agent has reached maximum ticket limit';
	END IF;

	UPDATE tickets
	SET assigned_to = p_agent_id,
		status_id = (SELECT id FROM ticket_statuses WHERE name = 'in_progress'),
		assignment_justification = COALESCE(NULLIF(BTRIM(p_justification), ''), assignment_justification)
	WHERE id = p_ticket_id;

	INSERT INTO audit_logs (id, ticket_id, actor_id, action, new_value)
	VALUES (
		gen_random_uuid(),
		p_ticket_id,
		p_actor_id,
		'assigned',
		jsonb_build_object(
			'agent_id', p_agent_id,
			'assignment_justification', COALESCE(NULLIF(BTRIM(p_justification), ''), 'Assigned manually.')
		)
	);
END;
$$;
