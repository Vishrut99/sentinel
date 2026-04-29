-- ============================================================
-- MIGRATION: v1 → v2
-- Run this ONLY if you already ran the old schema.
-- This script only makes the changes — does not recreate
-- anything that already exists.
-- ============================================================


-- ------------------------------------------------------------
-- STEP 1: Remove the junction tables that are no longer needed
-- ------------------------------------------------------------

DROP TABLE IF EXISTS ticket_kb_links;
-- removing the link table between tickets and knowledge articles

DROP TABLE IF EXISTS knowledge_articles;
-- removing knowledge articles (phase 2 feature, not needed now)

DROP TABLE IF EXISTS user_roles;
-- removing the junction table — role is moving directly onto users


-- ------------------------------------------------------------
-- STEP 2: Remove the standalone roles table
-- (was only needed to support user_roles junction)
-- ------------------------------------------------------------

DROP TABLE IF EXISTS roles;


-- ------------------------------------------------------------
-- STEP 3: Add role column directly to users
-- Simple CHECK constraint replaces the old roles table
-- ------------------------------------------------------------

ALTER TABLE users
    ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user'
    CHECK (role IN ('admin', 'agent', 'user'));

CREATE INDEX idx_users_role ON users (role);


-- ------------------------------------------------------------
-- STEP 4: Add the 3 stored procedures
-- (triggers and functions from v1 stay exactly as they are)
-- ------------------------------------------------------------

CREATE OR REPLACE PROCEDURE sp_assign_ticket(
    p_ticket_id  UUID,
    p_agent_id   UUID,
    p_actor_id   UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_ticket_status   SMALLINT;
    v_agent_available BOOLEAN;
    v_agent_max       INTEGER;
    v_agent_current   INTEGER;
BEGIN
    SELECT status_id
    INTO   v_ticket_status
    FROM   tickets
    WHERE  id = p_ticket_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    IF v_ticket_status IN (3, 4, 5) THEN
        RAISE EXCEPTION 'Cannot assign a ticket that is resolved, closed, or cancelled';
    END IF;

    SELECT is_available, max_tickets
    INTO   v_agent_available, v_agent_max
    FROM   agents
    WHERE  id = p_agent_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Agent % does not exist', p_agent_id;
    END IF;

    IF NOT v_agent_available THEN
        RAISE EXCEPTION 'Agent % is not currently available', p_agent_id;
    END IF;

    SELECT COUNT(*)
    INTO   v_agent_current
    FROM   tickets
    WHERE  assigned_to = p_agent_id
    AND    status_id IN (1, 2);

    IF v_agent_current >= v_agent_max THEN
        RAISE EXCEPTION 'Agent % has reached their maximum ticket limit (% tickets)',
            p_agent_id, v_agent_max;
    END IF;

    UPDATE tickets
    SET    assigned_to = p_agent_id,
           status_id   = 2
    WHERE  id = p_ticket_id;

    INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
    VALUES (
        p_ticket_id,
        p_actor_id,
        'assigned',
        jsonb_build_object('assigned_to', NULL),
        jsonb_build_object('assigned_to', p_agent_id)
    );
END;
$$;


CREATE OR REPLACE PROCEDURE sp_resolve_ticket(
    p_ticket_id UUID,
    p_actor_id  UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_status_id SMALLINT;
    v_due_at    TIMESTAMPTZ;
    v_breached  BOOLEAN := FALSE;
BEGIN
    SELECT status_id, due_at
    INTO   v_status_id, v_due_at
    FROM   tickets
    WHERE  id = p_ticket_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    IF v_status_id NOT IN (1, 2) THEN
        RAISE EXCEPTION 'Ticket must be open or in_progress to be resolved';
    END IF;

    UPDATE tickets
    SET    status_id = 3
    WHERE  id = p_ticket_id;

    IF v_due_at IS NOT NULL AND NOW() > v_due_at THEN
        v_breached := TRUE;
        UPDATE tickets
        SET    sla_breached = TRUE
        WHERE  id = p_ticket_id;
    END IF;

    INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
    VALUES (
        p_ticket_id,
        p_actor_id,
        'resolved',
        jsonb_build_object('status_id', v_status_id),
        jsonb_build_object(
            'status_id',    3,
            'sla_breached', v_breached,
            'resolved_at',  NOW()
        )
    );
END;
$$;


CREATE OR REPLACE PROCEDURE sp_register_agent(
    p_user_id    UUID,
    p_department VARCHAR(100),
    p_actor_id   UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_current_role VARCHAR(20);
    v_new_agent_id UUID;
BEGIN
    SELECT role
    INTO   v_current_role
    FROM   users
    WHERE  id = p_user_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;

    IF v_current_role IN ('agent', 'admin') THEN
        RAISE EXCEPTION 'User % is already an agent or admin', p_user_id;
    END IF;

    UPDATE users
    SET    role = 'agent'
    WHERE  id = p_user_id;

    v_new_agent_id := gen_random_uuid();

    INSERT INTO agents (id, user_id, department, is_available, max_tickets)
    VALUES (v_new_agent_id, p_user_id, p_department, TRUE, 10);

    RAISE NOTICE 'User % promoted to agent by %. New agent id: %',
        p_user_id, p_actor_id, v_new_agent_id;
END;
$$;


-- ============================================================
-- DONE. Changes made:
--   DROPPED  → ticket_kb_links, knowledge_articles, user_roles, roles
--   ADDED    → users.role column with CHECK constraint
--   ADDED    → idx_users_role index
--   ADDED    → sp_assign_ticket procedure
--   ADDED    → sp_resolve_ticket procedure
--   ADDED    → sp_register_agent procedure
-- Everything else (triggers, functions, tables) unchanged.
-- ============================================================
