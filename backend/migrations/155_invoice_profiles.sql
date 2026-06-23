CREATE TABLE IF NOT EXISTS invoice_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    invoice_title TEXT NOT NULL DEFAULT '',
    tax_no TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO invoice_profiles (user_id, invoice_title, tax_no, created_at, updated_at)
SELECT DISTINCT ON (user_id)
    user_id,
    invoice_title,
    tax_no,
    created_at,
    updated_at
FROM invoice_requests
ORDER BY user_id, created_at DESC, id DESC
ON CONFLICT (user_id) DO NOTHING;
