ALTER TABLE app_preferences
  ADD COLUMN IF NOT EXISTS run_mode TEXT NOT NULL DEFAULT 'localDocker';
