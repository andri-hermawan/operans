DELETE FROM core.report_schedules WHERE report_type = 'ritase_by_customer';
ALTER TABLE core.report_schedules DROP CONSTRAINT ck_report_schedules_type;
ALTER TABLE core.report_schedules ADD CONSTRAINT ck_report_schedules_type
    CHECK (report_type IN ('alert_summary', 'executive_hauling', 'executive_emplacement'));
