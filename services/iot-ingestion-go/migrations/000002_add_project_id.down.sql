-- 000002_add_project_id (DOWN)
DROP INDEX IF EXISTS iot.idx_telemetry_project_time;
DROP INDEX IF EXISTS iot.idx_devices_project;
ALTER TABLE iot.telemetry_events DROP COLUMN IF EXISTS project_id;
ALTER TABLE iot.devices          DROP COLUMN IF EXISTS project_id;
