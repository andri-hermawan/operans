DO $$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['core.projects', 'core.shifts', 'core.equipment', 'core.fuel_calibrations',
                             'core.fuel_calibration_points', 'core.operator_assignments', 'core.equipment_breakdowns',
                             'core.alert_types', 'core.alert_rules', 'core.alert_rule_recipients', 'auth.users',
                             'auth.project_members', 'geo.zones']
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_notify_master_changed ON %s', t);
    END LOOP;
END
$$;
DROP FUNCTION IF EXISTS core.notify_master_changed();

ALTER FUNCTION core.effective_alert_rules(uuid, text) SECURITY INVOKER RESET search_path;
ALTER FUNCTION geo.find_zone(uuid, double precision, double precision, timestamptz) SECURITY INVOKER RESET search_path;

DROP VIEW IF EXISTS geo.v_iot_zones;
DROP VIEW IF EXISTS core.v_iot_alert_recipients;
DROP VIEW IF EXISTS core.v_iot_equipment_breakdowns;
DROP VIEW IF EXISTS core.v_iot_operator_assignments;
DROP VIEW IF EXISTS core.v_iot_fuel_calibration_points;
DROP VIEW IF EXISTS core.v_iot_equipment;
DROP VIEW IF EXISTS core.v_iot_shifts;
DROP VIEW IF EXISTS core.v_iot_projects;
