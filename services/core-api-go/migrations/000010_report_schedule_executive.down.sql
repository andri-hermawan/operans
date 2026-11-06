DELETE FROM core.report_schedules WHERE report_type <> 'alert_summary';
ALTER TABLE core.report_schedules DROP CONSTRAINT ck_report_schedules_type;
ALTER TABLE core.report_schedules ADD CONSTRAINT ck_report_schedules_type CHECK (report_type IN ('alert_summary'));
