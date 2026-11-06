-- 000002_add_project_id (UP)
-- ADR 0002: data dibatasi per project. project_id merujuk core.projects (dibuat core-api nanti),
-- jadi foreign key & NOT NULL ditambahkan di migration berikutnya setelah tabel itu ada.

-- Alat dipasang di project mana
ALTER TABLE iot.devices ADD COLUMN project_id uuid;

-- Project SAAT kejadian (disalin dari devices ketika data masuk, sama seperti equipment_id)
ALTER TABLE iot.telemetry_events ADD COLUMN project_id uuid;

CREATE INDEX idx_devices_project          ON iot.devices (project_id);
CREATE INDEX idx_telemetry_project_time   ON iot.telemetry_events (project_id, event_time DESC);

COMMENT ON COLUMN iot.telemetry_events.project_id IS 'Project saat kejadian; dasar pembatasan akses per project.';
