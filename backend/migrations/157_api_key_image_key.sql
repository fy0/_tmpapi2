ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS is_image_key BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN api_keys.is_image_key IS 'Whether this user API key is selected for [img-key] image generation placeholder';

CREATE UNIQUE INDEX IF NOT EXISTS api_keys_user_image_key_unique
    ON api_keys (user_id)
    WHERE is_image_key = true AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS api_keys_user_image_key_lookup
    ON api_keys (user_id, created_at, id)
    WHERE deleted_at IS NULL;
