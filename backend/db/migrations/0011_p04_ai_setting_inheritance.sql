-- P04: distinguish a per-conversation choice from the catalog defaults.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS ai_settings_overridden BOOLEAN NOT NULL DEFAULT FALSE;
