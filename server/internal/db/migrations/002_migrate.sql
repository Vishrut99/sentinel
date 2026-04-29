INSERT INTO ticket_statuses (name, code)
VALUES
	('open', 'OPEN'),
	('in_progress', 'IN_PROGRESS'),
	('resolved', 'RESOLVED'),
	('closed', 'CLOSED'),
	('cancelled', 'CANCELLED')
ON CONFLICT (name) DO NOTHING;

INSERT INTO priorities (name, sla_response_hrs, sla_resolve_hrs)
VALUES
	('low', 8, 72),
	('medium', 4, 24),
	('high', 1, 8),
	('critical', 1, 4)
ON CONFLICT (name) DO NOTHING;

INSERT INTO categories (id, name, description)
VALUES (
	'00000000-0000-0000-0000-000000000001',
	'General',
	'Default category'
)
ON CONFLICT (id) DO NOTHING;

CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1;

CREATE OR REPLACE FUNCTION fn_set_ticket_number()
RETURNS trigger AS $$
BEGIN
	IF NEW.ticket_number IS NULL OR NEW.ticket_number = '' THEN
		NEW.ticket_number := 'TKT-' || LPAD(nextval('ticket_number_seq')::text, 6, '0');
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_set_ticket_due_at()
RETURNS trigger AS $$
DECLARE
	v_sla_resolve_hrs INTEGER;
BEGIN
	IF NEW.due_at IS NULL THEN
		SELECT p.sla_resolve_hrs
		INTO v_sla_resolve_hrs
		FROM priorities p
		WHERE p.id = NEW.priority_id;

		IF v_sla_resolve_hrs IS NOT NULL THEN
			NEW.due_at := NOW() + make_interval(hours => v_sla_resolve_hrs);
		END IF;
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_set_resolved_at()
RETURNS trigger AS $$
DECLARE
	v_resolved_status_id SMALLINT;
BEGIN
	SELECT id INTO v_resolved_status_id
	FROM ticket_statuses
	WHERE name = 'resolved';

	IF NEW.status_id = v_resolved_status_id AND OLD.status_id <> v_resolved_status_id THEN
		NEW.resolved_at := NOW();
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS trigger AS $$
BEGIN
	NEW.updated_at := NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_ticket_number ON tickets;
CREATE TRIGGER trg_set_ticket_number
BEFORE INSERT ON tickets
FOR EACH ROW EXECUTE FUNCTION fn_set_ticket_number();

DROP TRIGGER IF EXISTS trg_set_ticket_due_at ON tickets;
CREATE TRIGGER trg_set_ticket_due_at
BEFORE INSERT ON tickets
FOR EACH ROW EXECUTE FUNCTION fn_set_ticket_due_at();

DROP TRIGGER IF EXISTS trg_set_resolved_at ON tickets;
CREATE TRIGGER trg_set_resolved_at
BEFORE UPDATE ON tickets
FOR EACH ROW EXECUTE FUNCTION fn_set_resolved_at();

DROP TRIGGER IF EXISTS trg_tickets_updated_at ON tickets;
CREATE TRIGGER trg_tickets_updated_at
BEFORE UPDATE ON tickets
FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
