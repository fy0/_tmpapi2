ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS responses_image_generation_redirect_group_id BIGINT;

COMMENT ON COLUMN groups.responses_image_generation_redirect_group_id IS 'OpenAI Responses image_generation 请求重定向使用的 OpenAI 图片分组 ID';

UPDATE channels
SET features_config = features_config - 'responses_image_generation_redirect'
WHERE features_config ? 'responses_image_generation_redirect';
