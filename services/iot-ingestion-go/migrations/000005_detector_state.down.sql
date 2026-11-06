ALTER TABLE iot.fuel_events DROP CONSTRAINT IF EXISTS fk_fuel_events_alert, DROP COLUMN IF EXISTS alert_id;
ALTER TABLE iot.equipment_live_states DROP COLUMN IF EXISTS tracker;
