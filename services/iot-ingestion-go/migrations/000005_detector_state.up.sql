-- C3 (ADR 0009): status pendeteksi kejadian per unit + tautan kejadian fuel → alert.
--   tracker  = keadaan pendeteksi (awal berhenti, liter saat mesin mati, awal ngebut, …) dalam JSON.
--              Disimpan bersama live state agar kejadian yang sedang berjalan tidak hilang saat layanan restart.
--              Isinya internal layanan iot (domain.Tracker); boleh berubah bentuk tanpa migration.
--   alert_id = kejadian fuel `decrease` yang memicu alert fuel_decrease_engine_on/_off.
ALTER TABLE iot.equipment_live_states ADD COLUMN tracker jsonb NOT NULL DEFAULT '{}';
ALTER TABLE iot.fuel_events
    ADD COLUMN alert_id bigint,
    ADD CONSTRAINT fk_fuel_events_alert FOREIGN KEY (alert_id) REFERENCES iot.alerts (id);
