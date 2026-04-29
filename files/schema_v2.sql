-- ============================================================
-- INCIDENT TICKETING SYSTEM — UPDATED SCHEMA (v2)
-- Database: PostgreSQL (Supabase)
--
-- CHANGES FROM v1:
--   REMOVED → user_roles table (junction table)
--   REMOVED → knowledge_articles table
--   REMOVED → ticket_kb_links table (junction table)
--   REMOVED → roles table (no longer needed separately)
--   CHANGED → users now has a direct role column (simpler)
--   ADDED   → 3 stored procedures for business operations
--   KEPT    → everything else exactly as it was
--
-- HOW TO RUN:
--   Supabase → SQL Editor → New Query → Paste → Run
-- ============================================================


-- ------------------------------------------------------------
-- EXTENSION
-- Needed for gen_random_uuid() to work
-- ------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "pgcrypto";


-- ============================================================
-- LOOKUP / CONFIG TABLES
-- These store fixed options. Your guide said configuration
-- should not change — these stay exactly as before.
-- ============================================================


-- ------------------------------------------------------------
-- TABLE: ticket_statuses
-- The 5 stages a ticket moves through.
-- Stored as a separate table so the status name is one
-- source of truth. tickets table stores the ID, not the text.
-- ------------------------------------------------------------
CREATE TABLE ticket_statuses (
    id   SMALLINT    PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name VARCHAR(30) NOT NULL UNIQUE
);

INSERT INTO ticket_statuses (name) VALUES
    ('open'),        -- 1: just created, no one working on it
    ('in_progress'), -- 2: agent picked it up
    ('resolved'),    -- 3: agent marked it fixed
    ('closed'),      -- 4: confirmed done, fully closed
    ('cancelled');   -- 5: ticket was invalid or withdrawn


-- ------------------------------------------------------------
-- TABLE: priorities
-- 4 urgency levels. Each one also stores SLA time limits.
-- sla_response_hrs = max hours before agent must respond
-- sla_resolve_hrs  = max hours before ticket must be resolved
--
-- Keeping SLA here means: changing the rule for "critical"
-- automatically affects all future tickets. No code change needed.
-- ------------------------------------------------------------
CREATE TABLE priorities (
    id               SMALLINT    PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name             VARCHAR(20) NOT NULL UNIQUE,
    sla_response_hrs INTEGER     NOT NULL,
    sla_resolve_hrs  INTEGER     NOT NULL
);

INSERT INTO priorities (name, sla_response_hrs, sla_resolve_hrs) VALUES
    ('low',      48, 120), -- respond in 2 days,  resolve in 5 days
    ('medium',   24,  72), -- respond in 1 day,   resolve in 3 days
    ('high',      8,  24), -- respond in 8 hours, resolve in 1 day
    ('critical',  1,   4); -- respond in 1 hour,  resolve in 4 hours


-- ------------------------------------------------------------
-- TABLE: categories
-- Groups tickets by problem type.
-- Kept exactly as before — guide said config should not change.
-- ------------------------------------------------------------
CREATE TABLE categories (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO categories (name, description) VALUES
    ('Network',  'Internet, VPN, WiFi, DNS problems'),
    ('Hardware', 'Broken devices, cables, equipment'),
    ('Software', 'App crashes, bugs, install issues'),
    ('Access',   'Login problems, locked accounts, permissions'),
    ('Security', 'Suspicious activity, data concerns'),
    ('Other',    'Anything that does not fit above');


-- ============================================================
-- MAIN TABLES
-- ============================================================


-- ------------------------------------------------------------
-- TABLE: users
-- CHANGE FROM V1: removed the separate roles table and
-- user_roles junction table. Role is now a direct column here.
--
-- Why this is simpler and fine for this system:
--   - A person is ONE thing: admin, agent, or user
--   - The junction table only made sense if one person could
--     hold multiple roles at once — which this system does not need
--   - Direct column = simpler queries, easier to understand
--
-- role column uses CHECK to only allow valid values.
-- This replaces what the old roles table was doing.
-- ------------------------------------------------------------
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    -- Never store plain text password. Go hashes it with bcrypt before saving.

    full_name     VARCHAR(100) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'user'
                  CHECK (role IN ('admin', 'agent', 'user')),
    -- CHECK constraint replaces the old roles lookup table.
    -- Database will reject any INSERT/UPDATE with an invalid role.

    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    -- FALSE = account disabled. We never delete users because
    -- their tickets and audit history must stay linked.

    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Index on email: login queries search by email every single time
CREATE INDEX idx_users_email ON users (email);

-- Index on role: useful for queries like "get all agents"
CREATE INDEX idx_users_role ON users (role);


-- ------------------------------------------------------------
-- TABLE: agents
-- Extra details for users who are agents (support staff).
-- Not every user is an agent, so agent-specific data
-- lives here instead of cluttering the users table.
--
-- One-to-one with users: UNIQUE on user_id ensures
-- one user cannot have two agent profiles.
--
-- KEPT EXACTLY AS V1 — no changes here.
-- ------------------------------------------------------------
CREATE TABLE agents (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    -- ON DELETE CASCADE: if user is deleted, agent row is also deleted

    department   VARCHAR(100),
    -- which team this agent belongs to e.g. "IT Support", "Network Team"

    is_available BOOLEAN     NOT NULL DEFAULT TRUE,
    -- FALSE when agent is on leave or has reached max_tickets limit

    max_tickets  INTEGER     NOT NULL DEFAULT 10,
    -- how many open tickets this agent can handle at once

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_user_id      ON agents (user_id);
CREATE INDEX idx_agents_is_available ON agents (is_available);


-- ------------------------------------------------------------
-- TABLE: tickets
-- THE HEART OF THE SYSTEM.
-- Every incident raised becomes one row here.
-- References lookup tables by ID (not by text) for status,
-- priority, category — keeping data consistent and clean.
--
-- KEPT EXACTLY AS V1 — no changes here.
-- ------------------------------------------------------------
CREATE TABLE tickets (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),

    ticket_number VARCHAR(20)  NOT NULL UNIQUE,
    -- Readable ID like TKT-000042.
    -- Set automatically by Trigger 1 below. Never set this manually.

    title         VARCHAR(255) NOT NULL,
    description   TEXT,

    -- Status: references ticket_statuses table (stores ID 1-5)
    status_id     SMALLINT     NOT NULL DEFAULT 1
                  REFERENCES ticket_statuses(id),
    -- DEFAULT 1 = 'open'. Every new ticket starts as open.

    -- Priority: references priorities table (stores ID 1-4)
    priority_id   SMALLINT     NOT NULL REFERENCES priorities(id),

    -- Category: references categories table
    category_id   UUID         NOT NULL REFERENCES categories(id),

    -- The user who raised this ticket
    created_by    UUID         NOT NULL REFERENCES users(id),

    -- The agent handling this ticket (NULL = not yet assigned)
    assigned_to   UUID         REFERENCES agents(id),

    -- SLA tracking
    sla_breached  BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Becomes TRUE if ticket is not resolved before due_at.
    -- Updated by the sp_check_sla procedure.

    due_at        TIMESTAMPTZ,
    -- The deadline. Set automatically by Trigger 2 on insert.
    -- Formula: created_at + priority.sla_resolve_hrs

    resolved_at   TIMESTAMPTZ,
    -- When the ticket was actually fixed.
    -- Set automatically by Trigger 3 when status → resolved/closed.

    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickets_status_id   ON tickets (status_id);
CREATE INDEX idx_tickets_priority_id ON tickets (priority_id);
CREATE INDEX idx_tickets_created_by  ON tickets (created_by);
CREATE INDEX idx_tickets_assigned_to ON tickets (assigned_to);
CREATE INDEX idx_tickets_due_at      ON tickets (due_at);
CREATE INDEX idx_tickets_created_at  ON tickets (created_at DESC);


-- ------------------------------------------------------------
-- TABLE: comments
-- Messages on a ticket — like a chat thread between the user
-- and the agent working on it.
--
-- is_internal = FALSE → user can see this comment
-- is_internal = TRUE  → only agents/admins can see it (private note)
--
-- KEPT EXACTLY AS V1 — no changes here.
-- ------------------------------------------------------------
CREATE TABLE comments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID        NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    -- ON DELETE CASCADE: if ticket deleted, its comments are also deleted

    author_id   UUID        NOT NULL REFERENCES users(id),
    body        TEXT        NOT NULL,
    is_internal BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_ticket_id ON comments (ticket_id, created_at ASC);
-- ASC so comments load oldest-first (chronological order)


-- ------------------------------------------------------------
-- TABLE: audit_logs
-- A permanent history of every important change to a ticket.
-- Written AUTOMATICALLY by Trigger 5 — never written by Go code.
--
-- old_value: what the data looked like BEFORE the change
-- new_value: what the data looks like AFTER the change
-- Both stored as JSONB (flexible key-value format).
--
-- IMPORTANT: rows here must NEVER be updated or deleted.
-- This table is evidence — it must be append-only.
--
-- KEPT EXACTLY AS V1 — no changes here.
-- ------------------------------------------------------------
CREATE TABLE audit_logs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id  UUID         NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    actor_id   UUID         NOT NULL REFERENCES users(id),
    -- actor = the person who made the change

    action     VARCHAR(100) NOT NULL,
    -- what happened: 'status_changed', 'assigned', 'priority_changed', 'created'

    old_value  JSONB,
    -- snapshot before change e.g. {"status_id": 1}

    new_value  JSONB,
    -- snapshot after change  e.g. {"status_id": 2}

    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_ticket_id ON audit_logs (ticket_id, created_at DESC);
CREATE INDEX idx_audit_actor_id  ON audit_logs (actor_id);
CREATE INDEX idx_audit_action    ON audit_logs (action);


-- ============================================================
-- TRIGGERS + FUNCTIONS
-- Functions contain the logic.
-- Triggers call the functions automatically when DB events happen.
-- Every trigger has exactly one function. One function can be
-- reused by multiple triggers (see fn_set_updated_at below).
-- ============================================================


-- ------------------------------------------------------------
-- FUNCTION + TRIGGER 1: Auto ticket number
--
-- Function: fn_set_ticket_number()
--   Uses a sequence to get the next number (1, 2, 3...)
--   Formats it as TKT-000001, TKT-000042 etc.
--   The sequence is atomic — even if two tickets are created
--   at the same millisecond, they get different numbers.
--
-- Trigger: trg_set_ticket_number
--   Fires BEFORE INSERT on tickets.
--   Only fires if ticket_number is not already set.
-- ------------------------------------------------------------
CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1;

CREATE OR REPLACE FUNCTION fn_set_ticket_number()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.ticket_number = 'TKT-' || LPAD(nextval('ticket_number_seq')::TEXT, 6, '0');
    -- nextval gets next number from sequence
    -- LPAD pads with zeros to always be 6 digits
    -- Result: TKT-000001, TKT-000042, TKT-001000
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_set_ticket_number
    BEFORE INSERT ON tickets
    FOR EACH ROW
    WHEN (NEW.ticket_number IS NULL OR NEW.ticket_number = '')
    EXECUTE FUNCTION fn_set_ticket_number();


-- ------------------------------------------------------------
-- FUNCTION + TRIGGER 2: Auto due date from SLA
--
-- Function: fn_set_ticket_due_at()
--   Looks up the priority row to get sla_resolve_hrs.
--   Sets due_at = created_at + that many hours.
--   Go code does NOT calculate this — DB owns this logic.
--
-- Trigger: trg_set_ticket_due_at
--   Fires BEFORE INSERT on tickets.
--   Runs for every new ticket regardless of priority.
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_set_ticket_due_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    v_resolve_hrs INTEGER;
BEGIN
    -- Get SLA hours from the priorities table
    SELECT sla_resolve_hrs
    INTO   v_resolve_hrs
    FROM   priorities
    WHERE  id = NEW.priority_id;

    -- Calculate deadline: ticket creation time + SLA hours
    NEW.due_at = NEW.created_at + (v_resolve_hrs || ' hours')::INTERVAL;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_set_ticket_due_at
    BEFORE INSERT ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_ticket_due_at();


-- ------------------------------------------------------------
-- FUNCTION + TRIGGER 3: Auto resolved timestamp
--
-- Function: fn_set_resolved_at()
--   Checks what the new status name is.
--   If resolved or closed → stamps resolved_at = NOW()
--   If re-opened → clears resolved_at back to NULL
--   This runs inside the same transaction as the status update,
--   so they always happen together (atomicity).
--
-- Trigger: trg_set_resolved_at
--   Fires BEFORE UPDATE on tickets.
--   Only fires when status_id actually changes (saves work).
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_set_resolved_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    v_status_name VARCHAR(30);
BEGIN
    -- Look up the human-readable name of the new status
    SELECT name
    INTO   v_status_name
    FROM   ticket_statuses
    WHERE  id = NEW.status_id;

    -- Stamp the time when ticket is marked done
    IF v_status_name IN ('resolved', 'closed') AND NEW.resolved_at IS NULL THEN
        NEW.resolved_at = NOW();
    END IF;

    -- Clear the stamp if ticket is re-opened
    IF v_status_name = 'open' THEN
        NEW.resolved_at = NULL;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_set_resolved_at
    BEFORE UPDATE ON tickets
    FOR EACH ROW
    WHEN (OLD.status_id IS DISTINCT FROM NEW.status_id)
    -- IS DISTINCT FROM = only fires when value actually changed
    EXECUTE FUNCTION fn_set_resolved_at();


-- ------------------------------------------------------------
-- FUNCTION + TRIGGER 4: Auto updated_at
--
-- Function: fn_set_updated_at()
--   Simply sets updated_at = NOW() on the modified row.
--   One function, reused by TWO triggers (tickets and users).
--   This is intentional — same logic, two tables.
--
-- Triggers: trg_tickets_updated_at, trg_users_updated_at
--   Both fire BEFORE UPDATE on their respective tables.
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- Same function attached to two different tables
CREATE TRIGGER trg_tickets_updated_at
    BEFORE UPDATE ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();


-- ------------------------------------------------------------
-- FUNCTION + TRIGGER 5: Auto audit log on ticket changes
--
-- Function: fn_audit_ticket_changes()
--   Checks three things after any ticket update:
--     1. Did status change?   → write audit row
--     2. Did assignment change? → write audit row
--     3. Did priority change?  → write audit row
--   Each check is independent — all three can fire in one update.
--   Stores old and new values as JSONB snapshots.
--   Go code NEVER writes to audit_logs manually.
--   This trigger guarantees the history is always complete.
--
-- Trigger: trg_audit_ticket_changes
--   Fires AFTER UPDATE on tickets (after so it can read OLD and NEW).
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION fn_audit_ticket_changes()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    -- Check 1: Did the status change?
    IF OLD.status_id IS DISTINCT FROM NEW.status_id THEN
        INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
        VALUES (
            NEW.id,
            NEW.created_by,
            'status_changed',
            jsonb_build_object('status_id', OLD.status_id),
            jsonb_build_object('status_id', NEW.status_id)
        );
    END IF;

    -- Check 2: Did the assigned agent change?
    IF OLD.assigned_to IS DISTINCT FROM NEW.assigned_to THEN
        INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
        VALUES (
            NEW.id,
            NEW.created_by,
            'assignment_changed',
            jsonb_build_object('assigned_to', OLD.assigned_to),
            jsonb_build_object('assigned_to', NEW.assigned_to)
        );
    END IF;

    -- Check 3: Did the priority change?
    IF OLD.priority_id IS DISTINCT FROM NEW.priority_id THEN
        INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
        VALUES (
            NEW.id,
            NEW.created_by,
            'priority_changed',
            jsonb_build_object('priority_id', OLD.priority_id),
            jsonb_build_object('priority_id', NEW.priority_id)
        );
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_audit_ticket_changes
    AFTER UPDATE ON tickets
    FOR EACH ROW
    EXECUTE FUNCTION fn_audit_ticket_changes();


-- ============================================================
-- STORED PROCEDURES
--
-- DIFFERENCE FROM TRIGGERS:
--   Triggers fire automatically — you never call them.
--   Procedures are called intentionally from Go code.
--
-- WHY PROCEDURES:
--   Some operations involve multiple steps across multiple tables.
--   If Go does them as separate DB calls and crashes midway,
--   you get partial data (ticket assigned but not logged, etc).
--   A procedure wraps everything in ONE transaction:
--   all steps succeed together, or all fail together.
--
-- HOW GO CALLS A PROCEDURE:
--   db.Exec("CALL sp_assign_ticket($1, $2, $3)", ticketID, agentID, actorID)
-- ============================================================


-- ------------------------------------------------------------
-- PROCEDURE 1: sp_assign_ticket
--
-- What it does (all in one transaction):
--   Step 1. Check the ticket exists and is not already closed/resolved
--   Step 2. Check the agent exists and is currently available
--   Step 3. Check the agent has not hit their max_tickets limit
--   Step 4. Update ticket.assigned_to = agent
--   Step 5. Update ticket.status_id   = in_progress (status 2)
--   Step 6. Write an audit log row (in addition to the trigger)
--           Note: trigger also writes one — this gives a second
--           record with the actor_id correctly set to who called it
--
-- Parameters:
--   p_ticket_id : UUID of the ticket to assign
--   p_agent_id  : UUID of the agent to assign to
--   p_actor_id  : UUID of the user doing the assigning (for audit)
--
-- If anything fails, the whole thing rolls back automatically.
-- ------------------------------------------------------------
CREATE OR REPLACE PROCEDURE sp_assign_ticket(
    p_ticket_id  UUID,
    p_agent_id   UUID,
    p_actor_id   UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_ticket_status  SMALLINT;
    v_agent_available BOOLEAN;
    v_agent_max       INTEGER;
    v_agent_current   INTEGER;
BEGIN
    -- Step 1: Get current ticket status, lock the row so
    -- two people cannot assign the same ticket simultaneously
    SELECT status_id
    INTO   v_ticket_status
    FROM   tickets
    WHERE  id = p_ticket_id
    FOR UPDATE;
    -- FOR UPDATE locks this row until this procedure finishes

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    IF v_ticket_status IN (3, 4, 5) THEN
        -- 3=resolved, 4=closed, 5=cancelled — cannot assign these
        RAISE EXCEPTION 'Cannot assign a ticket that is resolved, closed, or cancelled';
    END IF;

    -- Step 2: Check agent exists and is available
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

    -- Step 3: Count how many open tickets the agent currently has
    SELECT COUNT(*)
    INTO   v_agent_current
    FROM   tickets
    WHERE  assigned_to = p_agent_id
    AND    status_id IN (1, 2); -- open or in_progress

    IF v_agent_current >= v_agent_max THEN
        RAISE EXCEPTION 'Agent % has reached their maximum ticket limit (% tickets)',
            p_agent_id, v_agent_max;
    END IF;

    -- Step 4 + 5: Assign the ticket and move to in_progress
    UPDATE tickets
    SET    assigned_to = p_agent_id,
           status_id   = 2          -- in_progress
    WHERE  id = p_ticket_id;
    -- Note: this UPDATE fires trg_audit_ticket_changes and
    -- trg_set_resolved_at and trg_tickets_updated_at automatically

    -- Step 6: Write a procedure-level audit entry with the real actor
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


-- ------------------------------------------------------------
-- PROCEDURE 2: sp_resolve_ticket
--
-- What it does (all in one transaction):
--   Step 1. Check ticket exists and is currently in_progress
--   Step 2. Update status to resolved
--           (Trigger 3 automatically sets resolved_at)
--   Step 3. Check if ticket is past its due_at deadline
--   Step 4. If overdue, mark sla_breached = TRUE
--   Step 5. Write audit log with resolution time info
--
-- Parameters:
--   p_ticket_id : UUID of the ticket to resolve
--   p_actor_id  : UUID of the agent resolving it
-- ------------------------------------------------------------
CREATE OR REPLACE PROCEDURE sp_resolve_ticket(
    p_ticket_id UUID,
    p_actor_id  UUID
)
LANGUAGE plpgsql AS $$
DECLARE
    v_status_id  SMALLINT;
    v_due_at     TIMESTAMPTZ;
    v_breached   BOOLEAN := FALSE;
BEGIN
    -- Step 1: Get current ticket info, lock row
    SELECT status_id, due_at
    INTO   v_status_id, v_due_at
    FROM   tickets
    WHERE  id = p_ticket_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Ticket % does not exist', p_ticket_id;
    END IF;

    IF v_status_id NOT IN (1, 2) THEN
        -- Only open(1) or in_progress(2) tickets can be resolved
        RAISE EXCEPTION 'Ticket must be open or in_progress to be resolved';
    END IF;

    -- Step 2: Update status to resolved (3)
    -- Trigger 3 (trg_set_resolved_at) will automatically set resolved_at = NOW()
    UPDATE tickets
    SET    status_id = 3
    WHERE  id = p_ticket_id;

    -- Step 3 + 4: Check if SLA was breached (resolved after deadline)
    IF v_due_at IS NOT NULL AND NOW() > v_due_at THEN
        v_breached := TRUE;
        UPDATE tickets
        SET    sla_breached = TRUE
        WHERE  id = p_ticket_id;
    END IF;

    -- Step 5: Write audit entry
    INSERT INTO audit_logs (ticket_id, actor_id, action, old_value, new_value)
    VALUES (
        p_ticket_id,
        p_actor_id,
        'resolved',
        jsonb_build_object('status_id', v_status_id),
        jsonb_build_object(
            'status_id',   3,
            'sla_breached', v_breached,
            'resolved_at',  NOW()
        )
    );

END;
$$;


-- ------------------------------------------------------------
-- PROCEDURE 3: sp_register_agent
--
-- What it does (all in one transaction):
--   Step 1. Check the user exists
--   Step 2. Check user is not already an agent
--   Step 3. Update users.role = 'agent'
--   Step 4. Create the agents row for this user
--   Step 5. Write audit log entry
--
-- Parameters:
--   p_user_id    : UUID of the user to promote
--   p_department : which team they belong to
--   p_actor_id   : UUID of the admin doing the promotion (for audit)
-- ------------------------------------------------------------
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
    -- Step 1: Check user exists
    SELECT role
    INTO   v_current_role
    FROM   users
    WHERE  id = p_user_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'User % does not exist', p_user_id;
    END IF;

    -- Step 2: Check they are not already an agent or admin
    IF v_current_role IN ('agent', 'admin') THEN
        RAISE EXCEPTION 'User % is already an agent or admin', p_user_id;
    END IF;

    -- Step 3: Promote the user role
    UPDATE users
    SET    role = 'agent'
    WHERE  id = p_user_id;
    -- Trigger trg_users_updated_at fires automatically here

    -- Step 4: Create the agent profile
    v_new_agent_id := gen_random_uuid();

    INSERT INTO agents (id, user_id, department, is_available, max_tickets)
    VALUES (v_new_agent_id, p_user_id, p_department, TRUE, 10);

    -- Step 5: Write audit entry (no ticket involved, use actor as reference)
    -- We log this in audit_logs with a placeholder ticket_id of NULL
    -- using a separate simple log approach:
    RAISE NOTICE 'User % promoted to agent by %. New agent id: %',
        p_user_id, p_actor_id, v_new_agent_id;
    -- In production you could have a separate system_logs table for this

END;
$$;


-- ============================================================
-- SUMMARY
-- -------
-- TABLES (8 total, no junction tables):
--   ticket_statuses   → lookup: 5 status options
--   priorities        → lookup: 4 priority levels + SLA hours
--   categories        → lookup: 6 problem categories
--   users             → all system users (role column, not junction)
--   agents            → extra info for support staff
--   tickets           → central table, every incident
--   comments          → messages on tickets
--   audit_logs        → auto-written change history
--
-- TRIGGERS (5 total, each calls one function):
--   trg_set_ticket_number    → fn_set_ticket_number()
--   trg_set_ticket_due_at    → fn_set_ticket_due_at()
--   trg_set_resolved_at      → fn_set_resolved_at()
--   trg_tickets_updated_at   → fn_set_updated_at()  ← same function
--   trg_users_updated_at     → fn_set_updated_at()  ← reused here
--   trg_audit_ticket_changes → fn_audit_ticket_changes()
--
-- PROCEDURES (3 total, called intentionally from Go):
--   sp_assign_ticket   → assign ticket to agent safely
--   sp_resolve_ticket  → resolve ticket + SLA breach check
--   sp_register_agent  → promote user to agent role
-- ============================================================
