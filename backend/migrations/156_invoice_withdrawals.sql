ALTER TABLE invoice_requests
    ADD COLUMN IF NOT EXISTS withdrawn_at TIMESTAMPTZ;

ALTER TABLE invoice_request_redeem_codes
    ADD COLUMN IF NOT EXISTS released_at TIMESTAMPTZ;

ALTER TABLE invoice_requests
    DROP CONSTRAINT IF EXISTS chk_invoice_requests_status;

ALTER TABLE invoice_requests
    ADD CONSTRAINT chk_invoice_requests_status
    CHECK (status IN ('pending', 'issued', 'rejected', 'withdrawn'));

ALTER TABLE invoice_request_redeem_codes
    DROP CONSTRAINT IF EXISTS uq_invoice_request_redeem_codes_redeem_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_invoice_request_redeem_codes_redeem_code_active
    ON invoice_request_redeem_codes(redeem_code_id)
    WHERE released_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_invoice_request_redeem_codes_released_at
    ON invoice_request_redeem_codes(released_at);
