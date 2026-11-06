-- B8 (ADR 0004/0005): data induk yang DIBACA layanan IoT, lewat view & fungsi saja (bukan tabel langsung),
-- + sinyal NOTIFY agar cache data induk di memori IoT diperbarui saat ada perubahan.
-- Role iot_ingest hanya mendapat SELECT pada view & EXECUTE pada fungsi di bawah.

-- =====================================================================
-- 1. View baca untuk IoT (schema core & geo; iot_ingest sudah punya USAGE)
-- =====================================================================
CREATE VIEW core.v_iot_projects AS
SELECT id AS project_id, project_code, project_name, timezone, is_active
  FROM core.projects;

CREATE VIEW core.v_iot_shifts AS
SELECT s.id AS shift_id, s.project_id, s.shift_code, s.shift_name, s.start_time, s.end_time, s.sequence, p.timezone
  FROM core.shifts s JOIN core.projects p ON p.id = s.project_id
 WHERE s.is_active;

CREATE VIEW core.v_iot_equipment AS
SELECT id AS equipment_id, project_id, equipment_code, equipment_alias, equipment_type, class, is_active
  FROM core.equipment;

CREATE VIEW core.v_iot_fuel_calibration_points AS
SELECT c.equipment_id, c.calibration_type, c.tank_capacity_liters, p.sensor_value, p.volume_liters
  FROM core.fuel_calibrations c JOIN core.fuel_calibration_points p ON p.equipment_id = c.equipment_id;

-- operator & breakdown harian: hanya 7 hari operasional terakhir (cukup untuk enrichment telemetry)
CREATE VIEW core.v_iot_operator_assignments AS
SELECT o.project_id, o.operational_date, s.shift_code, o.equipment_code, o.equipment_id, o.operator_name
  FROM core.operator_assignments o JOIN core.shifts s ON s.id = o.shift_id
 WHERE o.operational_date >= current_date - 7;

CREATE VIEW core.v_iot_equipment_breakdowns AS
SELECT b.project_id, b.operational_date, s.shift_code, b.equipment_code, b.equipment_id, b.status,
       b.category_code, b.started_at, b.ended_at
  FROM core.equipment_breakdowns b JOIN core.shifts s ON s.id = b.shift_id
 WHERE b.operational_date >= current_date - 7;

-- penerima WhatsApp: hanya data yang dibutuhkan (tanpa email/password)
CREATE VIEW core.v_iot_alert_recipients AS
SELECT id AS user_id, name, phone_number
  FROM auth.users
 WHERE is_active AND phone_number IS NOT NULL;

CREATE VIEW geo.v_iot_zones AS
SELECT id AS zone_id, project_id, source_fid, category, name, priority, area_m2, geom, valid_from, valid_to
  FROM geo.zones;

-- =====================================================================
-- 2. Fungsi yang dipakai IoT berjalan dengan hak pemiliknya (core_migrator),
--    sehingga iot_ingest tidak perlu hak baca tabel aturan/user/zona.
-- =====================================================================
ALTER FUNCTION core.effective_alert_rules(uuid, text) SECURITY DEFINER SET search_path = pg_catalog, public;
ALTER FUNCTION geo.find_zone(uuid, double precision, double precision, timestamptz)
      SECURITY DEFINER SET search_path = pg_catalog, public;

GRANT SELECT ON core.v_iot_projects, core.v_iot_shifts, core.v_iot_equipment, core.v_iot_fuel_calibration_points,
                core.v_iot_operator_assignments, core.v_iot_equipment_breakdowns, core.v_iot_alert_recipients,
                geo.v_iot_zones TO iot_ingest;
GRANT SELECT ON core.v_iot_projects, core.v_iot_shifts, core.v_iot_equipment, core.v_iot_fuel_calibration_points,
                core.v_iot_operator_assignments, core.v_iot_equipment_breakdowns, core.v_iot_alert_recipients,
                geo.v_iot_zones TO core_api;
GRANT EXECUTE ON FUNCTION core.effective_alert_rules(uuid, text) TO iot_ingest;
GRANT EXECUTE ON FUNCTION geo.find_zone(uuid, double precision, double precision, timestamptz) TO iot_ingest;

-- =====================================================================
-- 3. NOTIFY "simha_master_changed" saat data induk berubah. Payload = nama tabel; IoT cukup
--    memuat ulang bagian cache itu. Per perintah SQL (bukan per baris), dan PostgreSQL
--    menggabungkan payload sama dalam satu transaksi → impor ribuan baris = satu sinyal.
-- =====================================================================
CREATE FUNCTION core.notify_master_changed() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_notify('simha_master_changed', TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME);
    RETURN NULL;
END
$$;

DO $$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['core.projects', 'core.shifts', 'core.equipment', 'core.fuel_calibrations',
                             'core.fuel_calibration_points', 'core.operator_assignments', 'core.equipment_breakdowns',
                             'core.alert_types', 'core.alert_rules', 'core.alert_rule_recipients', 'auth.users',
                             'auth.project_members', 'geo.zones']
    LOOP
        EXECUTE format('CREATE TRIGGER trg_notify_master_changed AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON %s
                        FOR EACH STATEMENT EXECUTE FUNCTION core.notify_master_changed()', t);
    END LOOP;
END
$$;
