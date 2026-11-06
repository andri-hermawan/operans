-- 000003_standard_codes_and_derived_tables (DOWN)
DROP TABLE IF EXISTS iot.alerts;
DROP TABLE IF EXISTS iot.geofence_events;
DROP TABLE IF EXISTS iot.fuel_events;
DROP TABLE IF EXISTS iot.equipment_live_states;

DROP INDEX IF EXISTS iot.idx_telemetry_project_opdate;
ALTER TABLE iot.telemetry_events DROP COLUMN IF EXISTS operational_date;
ALTER TABLE iot.telemetry_events DROP COLUMN IF EXISTS shift_code;
ALTER TABLE iot.telemetry_events ADD COLUMN shift varchar(100);

-- Kembali ke CHECK lama. NOT VALID: baris yang sudah memakai kode baku tidak membuat rollback gagal.
ALTER TABLE iot.telemetry_events DROP CONSTRAINT ck_telemetry_vessel_status;
ALTER TABLE iot.telemetry_events DROP CONSTRAINT ck_telemetry_equipment_status;
ALTER TABLE iot.telemetry_events
    ADD CONSTRAINT ck_telemetry_vessel_status CHECK (vessel_status IN ('Loaded', 'Empty', 'Unknown')) NOT VALID,
    ADD CONSTRAINT ck_telemetry_equip_status  CHECK (equipment_status IN ('Running', 'Idle', 'Stop')) NOT VALID;
