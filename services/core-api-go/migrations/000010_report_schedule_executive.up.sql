-- Report Schedule: tambah dua jenis laporan terjadwal (Executive Summary Hauling & Emplacement).
ALTER TABLE core.report_schedules DROP CONSTRAINT ck_report_schedules_type;
ALTER TABLE core.report_schedules ADD CONSTRAINT ck_report_schedules_type
    CHECK (report_type IN ('alert_summary', 'executive_hauling', 'executive_emplacement'));
