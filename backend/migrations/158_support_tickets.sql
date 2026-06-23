CREATE TABLE IF NOT EXISTS support_tickets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    user_name VARCHAR(100) NOT NULL DEFAULT '',
    title VARCHAR(200) NOT NULL,
    category VARCHAR(40) NOT NULL DEFAULT 'feedback',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS support_ticket_messages (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_id BIGINT NULL,
    author_role VARCHAR(20) NOT NULL DEFAULT 'user',
    author_email VARCHAR(255) NOT NULL DEFAULT '',
    author_name VARCHAR(100) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_user_id ON support_tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets(status);
CREATE INDEX IF NOT EXISTS idx_support_tickets_category ON support_tickets(category);
CREATE INDEX IF NOT EXISTS idx_support_tickets_last_message_at ON support_tickets(last_message_at);
CREATE INDEX IF NOT EXISTS idx_support_tickets_created_at ON support_tickets(created_at);
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_last_message ON support_tickets(user_id, last_message_at);

CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_ticket_id ON support_ticket_messages(ticket_id);
CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_author_id ON support_ticket_messages(author_id);
CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_created_at ON support_ticket_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_ticket_created ON support_ticket_messages(ticket_id, created_at);

INSERT INTO settings (key, value, updated_at)
VALUES ('ticket_entry_visibility', 'all', NOW())
ON CONFLICT (key) DO NOTHING;
