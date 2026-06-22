CREATE TABLE IF NOT EXISTS invoice_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'pending',
    invoice_title TEXT NOT NULL,
    tax_no TEXT NOT NULL DEFAULT '',
    amount NUMERIC(20, 8) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    admin_note TEXT NOT NULL DEFAULT '',
    file_name TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    issued_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_invoice_requests_status CHECK (status IN ('pending', 'issued', 'rejected')),
    CONSTRAINT chk_invoice_requests_amount_positive CHECK (amount > 0)
);

CREATE TABLE IF NOT EXISTS invoice_request_redeem_codes (
    invoice_request_id BIGINT NOT NULL REFERENCES invoice_requests(id) ON DELETE CASCADE,
    redeem_code_id BIGINT NOT NULL REFERENCES redeem_codes(id) ON DELETE RESTRICT,
    amount NUMERIC(20, 8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (invoice_request_id, redeem_code_id),
    CONSTRAINT uq_invoice_request_redeem_codes_redeem_code UNIQUE (redeem_code_id),
    CONSTRAINT chk_invoice_request_redeem_codes_amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_invoice_requests_user_id ON invoice_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_status ON invoice_requests(status);
CREATE INDEX IF NOT EXISTS idx_invoice_requests_created_at ON invoice_requests(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_invoice_request_redeem_codes_invoice_id
    ON invoice_request_redeem_codes(invoice_request_id);
