-- Observation-only additive columns; do not derive or backfill historical IDs.
-- Keep legacy session_id and request_id unchanged.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS conversation_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS conversation_id_source VARCHAR(64),
    ADD COLUMN IF NOT EXISTS request_metadata JSONB;

ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS conversation_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS conversation_id_source VARCHAR(64),
    ADD COLUMN IF NOT EXISTS request_metadata JSONB;

COMMENT ON COLUMN usage_logs.conversation_id IS
    'Observation-only client correlation ID; scope by user_id and api_key_id; never a routing or authorization key';
COMMENT ON COLUMN ops_error_logs.conversation_id IS
    'Observation-only client correlation ID; scope by user_id and api_key_id; never a routing or authorization key';
