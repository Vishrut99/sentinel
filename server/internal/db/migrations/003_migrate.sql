CREATE OR REPLACE PROCEDURE sp_assign_ticket(
	p_ticket_id UUID,
	p_agent_id UUID,
	p_actor_id UUID
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
		status_id = (SELECT id FROM ticket_statuses WHERE name = 'in_progress')
	WHERE id = p_ticket_id;

	INSERT INTO audit_logs (id, ticket_id, actor_id, action, new_value)
	VALUES (
		gen_random_uuid(),
		p_ticket_id,
		p_actor_id,
		'assigned',
		jsonb_build_object('agent_id', p_agent_id)
	);
END;
$$;

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

DROP PROCEDURE IF EXISTS sp_register_agent(UUID, VARCHAR, UUID);

CREATE OR REPLACE PROCEDURE sp_register_agent(
	p_user_id UUID,
	p_department VARCHAR,
	p_max_tickets INT,
	p_actor_id UUID
)
LANGUAGE plpgsql
AS $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM users WHERE id = p_user_id) THEN
		RAISE EXCEPTION 'User not found';
	END IF;

	IF EXISTS (
		SELECT 1 FROM users
		WHERE id = p_user_id
		AND role IN ('agent', 'admin')
	) THEN
		RAISE EXCEPTION 'User is already an agent or admin';
	END IF;

	IF EXISTS (
		SELECT 1 FROM agents
		WHERE user_id = p_user_id
	) THEN
		RAISE EXCEPTION 'User is already an agent or admin';
	END IF;

	IF p_max_tickets IS NULL OR p_max_tickets < 1 THEN
		RAISE EXCEPTION 'max_tickets must be at least 1';
	END IF;

	UPDATE users
	SET role = 'agent'
	WHERE id = p_user_id;

	INSERT INTO agents (id, user_id, department, is_available, max_tickets)
	VALUES (gen_random_uuid(), p_user_id, NULLIF(BTRIM(p_department), ''), true, p_max_tickets);
END;
$$;
