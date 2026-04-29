CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	full_name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'user',
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
	id UUID PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ticket_statuses (
	id SMALLSERIAL PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	code TEXT
);

CREATE TABLE IF NOT EXISTS priorities (
	id SMALLSERIAL PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	sla_response_hrs INTEGER NOT NULL,
	sla_resolve_hrs INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS agents (
	id UUID PRIMARY KEY,
	user_id UUID NOT NULL UNIQUE REFERENCES users(id),
	department TEXT,
	is_available BOOLEAN NOT NULL DEFAULT true,
	max_tickets INTEGER NOT NULL DEFAULT 10,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tickets (
	id UUID PRIMARY KEY,
	ticket_number TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	description TEXT,
	ai_insights JSONB,
	status_id SMALLINT NOT NULL REFERENCES ticket_statuses(id),
	priority_id SMALLINT NOT NULL REFERENCES priorities(id),
	category_id UUID NOT NULL REFERENCES categories(id),
	created_by UUID NOT NULL REFERENCES users(id),
	assigned_to UUID REFERENCES agents(id),
	parent_id UUID REFERENCES tickets(id),
	is_problem BOOLEAN NOT NULL DEFAULT false,
	sla_breached BOOLEAN NOT NULL DEFAULT false,
	due_at TIMESTAMPTZ,
	resolved_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS comments (
	id UUID PRIMARY KEY,
	ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
	author_id UUID NOT NULL REFERENCES users(id),
	body TEXT NOT NULL,
	is_internal BOOLEAN NOT NULL DEFAULT false,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
	id UUID PRIMARY KEY,
	ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
	actor_id UUID NOT NULL REFERENCES users(id),
	action TEXT NOT NULL,
	old_value JSONB,
	new_value JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE ticket_statuses ADD COLUMN IF NOT EXISTS code TEXT;

ALTER TABLE tickets ADD COLUMN IF NOT EXISTS ticket_number TEXT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS ai_insights JSONB;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS parent_id UUID;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS is_problem BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS sla_breached BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS due_at TIMESTAMPTZ;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_tickets_created_by ON tickets(created_by);
CREATE INDEX IF NOT EXISTS idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX IF NOT EXISTS idx_tickets_status_id ON tickets(status_id);
CREATE INDEX IF NOT EXISTS idx_tickets_priority_id ON tickets(priority_id);
CREATE INDEX IF NOT EXISTS idx_tickets_parent_id ON tickets(parent_id);
CREATE INDEX IF NOT EXISTS idx_tickets_is_problem ON tickets(is_problem);
CREATE INDEX IF NOT EXISTS idx_comments_ticket_id ON comments(ticket_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ticket_id ON audit_logs(ticket_id);
