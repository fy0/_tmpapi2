-- Fork high water: 256; upstream v0.2.19 high water: 248.
-- Additive only. Never reinterpret historical rewrites as automatic retries.
ALTER TABLE error_passthrough_rules
    ADD COLUMN IF NOT EXISTS action VARCHAR(16) NOT NULL DEFAULT '';

UPDATE error_passthrough_rules
SET action = CASE
    WHEN passthrough_code AND passthrough_body THEN 'passthrough'
    ELSE 'rewrite'
END
WHERE action = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'error_passthrough_rules'::regclass
          AND conname = 'error_passthrough_rules_action_check'
    ) THEN
        ALTER TABLE error_passthrough_rules
            ADD CONSTRAINT error_passthrough_rules_action_check
            CHECK (action IN ('', 'passthrough', 'rewrite', 'retry'));
    END IF;
END $$;
