-- Fork high water before this feature: 256; upstream v0.2.19: 248.
-- Nontransactional, online index builds. No historical IDs are manufactured.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_conversation_scope
    ON usage_logs (conversation_id, user_id, api_key_id, created_at DESC, id DESC)
    WHERE conversation_id IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_error_logs_conversation_scope
    ON ops_error_logs (conversation_id, user_id, api_key_id, created_at DESC, id DESC)
    WHERE conversation_id IS NOT NULL;
