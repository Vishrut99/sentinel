-- ============================================================
-- MIGRATION: v2 → v3
-- Run this in Supabase SQL Editor if you already ran v2.
--
-- FIXES:
--   1. Drops the audit trigger — procedures handle audit now
--      (trigger was logging wrong actor_id and duplicating rows)
--   2. Adds `code` column to ticket_statuses so procedures
--      never rely on hardcoded IDs like IN (1,2,3)
--   3. Rewrites sp_assign_ticket and sp_resolve_ticket to use
--      status codes instead of hardcoded numbers
-- ============================================================


-- ------------------------------------------------------------
-- FIX 1: Remove the audit trigger and its function
-- Procedures sp_assign_ticket and sp_resolve_ticket already
-- write audit rows with the correct actor_id passed in.
-- The trigger was writing a second duplicate row using
-- NEW.created_by which is wrong — that's the ticket creator,
-- not the person making the change.
-- ------------------------------------------------------------

DROP TRIGGER  IF EXISTS trg_audit_ticket_changes ON tickets;
DROP FUNCTION IF EXISTS fn_audit_ticket_changes();


-- ------------------------------------------------------------
-- FIX 2: Add `code` column to ticket_statuses
-- Procedures will now look up status ID by code name
-- instead of hardcoding numbers like IN (1, 2, 3).
-- This means reordering seed data can never break procedures.
-- ------------------------------------------------------------

ALTER TABLE ticket_statuses
    ADD COLUMN IF NOT EXISTS code VARCHAR(30) UNIQUE;

-- Set code = name for all existing rows (they are the same values)
UPDATE ticket_statuses SET code = name;

-- Make it NOT NULL after filling existing rows
ALTER TABLE ticket_statuses
    ALTER COLUMN code SET NOT NULL;


-- ------------------------------------------------------------
-- FIX 3a: Rewrite sp_assign_ticket using status codes
-- Changes: IN (3,4,5) → look up by code name
--          status_id = 2 → look up 'in_progress' by code
-- ------------------------------------------------------------

CREATE OR REPLACE PROCEDURE sp_assign_ticket(
    p_ticket_id  UUID,
    p_agent_id   UUID,
    p_actor_id   UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_ticket_status_code  VARCHAR(30);
    v_agent_available     BOOLEAN;
    v_agent_max           INTEGER;
    v_agent_current       INTEGER;
    v_in_progress_id      SMALLINT;
BEGIN
    -- Get current status code of the ticket, lock the row
    SELECT ts.code
    INTO   v_ticket_status_code
    FROM   tickets t
    JOIN   ticket_statuses ts ON ts.id = t.status_id
    WHERE  t.id = p_ticket_id
    FOR UPDATE;
    -- FOR UPDATE locks the row so two people cannot assign simultaneously

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    -- Use code names, not IDs — safe even if seed order changes
    IF v_ticket_status_code IN ('resolved', 'closed', 'cancelled') THEN
        RAISE EXCEPTION 'Cannot assign a ticket that is resolved, closed, or cancelled';
    END IF;

    -- Check agent exists and is available
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

    -- Count active tickets this agent currently has
    SELECT COUNT(*)
    INTO   v_agent_current
    FROM   tickets t
    JOIN   ticket_statuses ts ON ts.id = t.status_id
    WHERE  t.assigned_to = p_agent_id
    AND    ts.code IN ('open', 'in_progress');

    IF v_agent_current >= v_agent_max THEN
        RAISE EXCEPTION 'Agent % has reached their maximum ticket limit (% tickets)',
            p_agent_id, v_agent_max;
    END IF;

    -- Look up the in_progress status ID by code
    SELECT id INTO v_in_progress_id
    FROM   ticket_statuses
    WHERE  code = 'in_progress';

    -- Assign ticket and move to in_progress
    UPDATE tickets
    SET    assigned_to = p_agent_id,
           status_id   = v_in_progress_id
    WHERE  id = p_ticket_id;
    -- trg_set_resolved_at and trg_tickets_updated_at still fire automatically

    -- Write audit log with the correct actor (the person doing the assigning)
    INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
    VALUES (
        p_ticket_id,
        p_actor_id,
        'assigned',
        jsonb_build_object('assigned_to', NULL, 'status', 'open'),
        jsonb_build_object('assigned_to', p_agent_id, 'status', 'in_progress')
    );
END;
$$;


-- ------------------------------------------------------------
-- FIX 3b: Rewrite sp_resolve_ticket using status codes
-- Changes: status_id NOT IN (1,2) → check by code name
--          status_id = 3 → look up 'resolved' by code
-- ------------------------------------------------------------

CREATE OR REPLACE PROCEDURE sp_resolve_ticket(
    p_ticket_id UUID,
    p_actor_id  UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_status_code   VARCHAR(30);
    v_due_at        TIMESTAMPTZ;
    v_breached      BOOLEAN := FALSE;
    v_resolved_id   SMALLINT;
BEGIN
    -- Get current status code, lock row
    SELECT ts.code, t.due_at
    INTO   v_status_code, v_due_at
    FROM   tickets t
    JOIN   ticket_statuses ts ON ts.id = t.status_id
    WHERE  t.id = p_ticket_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    -- Only open or in_progress tickets can be resolved
    IF v_status_code NOT IN ('open', 'in_progress') THEN
        RAISE EXCEPTION 'Ticket must be open or in_progress to be resolved. Current status: %',
            v_status_code;
    END IF;

    -- Look up resolved status ID by code
    SELECT id INTO v_resolved_id
    FROM   ticket_statuses
    WHERE  code = 'resolved';

    -- Update to resolved
    -- trg_set_resolved_at fires automatically and sets resolved_at = NOW()
    UPDATE tickets
    SET    status_id = v_resolved_id
    WHERE  id = p_ticket_id;

    -- Check if SLA was breached (resolved after the deadline)
    IF v_due_at IS NOT NULL AND NOW() > v_due_at THEN
        v_breached := TRUE;
        UPDATE tickets
        SET    sla_breached = TRUE
        WHERE  id = p_ticket_id;
    END IF;

    -- Write audit log with correct actor
    INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
    VALUES (
        p_ticket_id,
        p_actor_id,
        'resolved',
        jsonb_build_object('status', v_status_code),
        jsonb_build_object(
            'status',       'resolved',
            'sla_breached', v_breached,
            'resolved_at',  NOW()
        )
    );
END;
$$;


-- ============================================================
-- DONE. Summary of changes:
--
-- DROPPED  → trg_audit_ticket_changes trigger
-- DROPPED  → fn_audit_ticket_changes() function
-- ADDED    → ticket_statuses.code column (filled from name)
-- REWRITTEN → sp_assign_ticket (uses code not IDs)
-- REWRITTEN → sp_resolve_ticket (uses code not IDs)
--
-- sp_register_agent has no status ID checks so no change needed.
-- All other triggers (ticket number, due_at, resolved_at,
-- updated_at) are unchanged and still active.
-- ============================================================
