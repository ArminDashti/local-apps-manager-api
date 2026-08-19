ALTER TABLE app_preferences
  ADD COLUMN IF NOT EXISTS local_enabled BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE app_preferences
  ADD COLUMN IF NOT EXISTS docker_enabled BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE app_preferences
  ADD COLUMN IF NOT EXISTS public_enabled BOOLEAN NOT NULL DEFAULT false;

UPDATE app_preferences
SET
  local_enabled = enabled AND run_mode = 'local',
  docker_enabled = enabled AND run_mode = 'localDocker',
  public_enabled = enabled AND run_mode = 'server'
WHERE enabled = true;
