ALTER TABLE iot.equipment_live_states
    DROP CONSTRAINT IF EXISTS ck_live_states_last_inside_vessel,
    DROP COLUMN IF EXISTS last_inside_vessel_status,
    DROP COLUMN IF EXISTS last_inside_fid,
    DROP COLUMN IF EXISTS zone_id;
ALTER TABLE iot.geofence_events  DROP COLUMN IF EXISTS zone_id;
ALTER TABLE iot.telemetry_events DROP COLUMN IF EXISTS zone_id;
