-- B7 (ADR 0008): kalibrasi fuel per equipment + jenis, aturan & penerima alert.
-- Lama: fuel_calibrations, alert_categories, alert_rules, alert_rule_recipients.

-- =====================================================================
-- 1. Kalibrasi fuel: kepala (satu per equipment) + titik nilai sensor → liter
-- =====================================================================
CREATE TABLE core.fuel_calibrations (
    equipment_id          uuid           NOT NULL,
    calibration_type      varchar(20)    NOT NULL,   -- lls | percentage (lama: LLS / PERCENTAGE)
    tank_capacity_liters  numeric(8,2)   NOT NULL,   -- = liter titik terakhir
    max_sensor_value      numeric(10,2)  NOT NULL,   -- = nilai sensor titik terakhir (LLS mis. 4095; ECU 100)
    is_linear             boolean        NOT NULL,   -- true = dibangkitkan tiap 5 liter; false = tabel uji isi tangki
    created_at            timestamptz    NOT NULL DEFAULT now(),
    created_by            uuid,
    updated_at            timestamptz    NOT NULL DEFAULT now(),
    updated_by            uuid,

    CONSTRAINT pk_fuel_calibrations            PRIMARY KEY (equipment_id),
    CONSTRAINT fk_fuel_calibrations_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_fuel_calibrations_type       CHECK (calibration_type IN ('lls', 'percentage')),
    CONSTRAINT ck_fuel_calibrations_values     CHECK (tank_capacity_liters > 0 AND max_sensor_value > 0)
);

CREATE TABLE core.fuel_calibration_points (
    equipment_id   uuid           NOT NULL,
    sensor_value   numeric(10,2)  NOT NULL,   -- lama: fuel_level
    volume_liters  numeric(8,2)   NOT NULL,   -- lama: fuel_volume

    CONSTRAINT pk_fuel_calibration_points       PRIMARY KEY (equipment_id, sensor_value),
    CONSTRAINT fk_fuel_calibration_points_head  FOREIGN KEY (equipment_id)
                                                REFERENCES core.fuel_calibrations (equipment_id) ON DELETE CASCADE,
    CONSTRAINT ck_fuel_calibration_points       CHECK (sensor_value >= 0 AND volume_liters >= 0)
);

-- =====================================================================
-- 2. Jenis alert + nilai bawaan (lama: alert_categories + aturan global)
-- =====================================================================
CREATE TABLE core.alert_types (
    code                       varchar(40)    NOT NULL,   -- sama dengan iot.alerts.alert_type
    legacy_code                varchar(10)    NOT NULL,   -- OFT, OVS, … (sistem lama)
    name                       varchar(100)   NOT NULL,
    description                varchar(255)   NOT NULL,
    uses_duration              boolean        NOT NULL,
    uses_speed_limit           boolean        NOT NULL,
    uses_fuel_drop             boolean        NOT NULL,
    default_duration_minutes   numeric(6,2),
    default_speed_limit_kph    numeric(6,2),
    default_fuel_drop_liters   numeric(8,2),
    sort_order                 smallint       NOT NULL,
    updated_at                 timestamptz    NOT NULL DEFAULT now(),
    updated_by                 uuid,

    CONSTRAINT pk_alert_types        PRIMARY KEY (code),
    CONSTRAINT uq_alert_types_legacy UNIQUE (legacy_code),
    CONSTRAINT ck_alert_types_code   CHECK (code ~ '^[a-z][a-z0-9_]*$'),
    -- nilai bawaan wajib ada untuk parameter yang dipakai, dan kosong untuk yang tidak dipakai
    CONSTRAINT ck_alert_types_params CHECK (
        (uses_duration    = (default_duration_minutes IS NOT NULL)) AND
        (uses_speed_limit = (default_speed_limit_kph  IS NOT NULL)) AND
        (uses_fuel_drop   = (default_fuel_drop_liters IS NOT NULL))),
    CONSTRAINT ck_alert_types_values CHECK (
        coalesce(default_duration_minutes, 0) >= 0 AND coalesce(default_speed_limit_kph, 0) >= 0
        AND coalesce(default_fuel_drop_liters, 0) >= 0)
);

-- Nilai bawaan = aturan global sistem lama (sama dengan isi fms_lite.alert_rules)
INSERT INTO core.alert_types (code, legacy_code, name, description, uses_duration, uses_speed_limit, uses_fuel_drop,
                              default_duration_minutes, default_speed_limit_kph, default_fuel_drop_liters, sort_order) VALUES
    ('off_track', 'OFT', 'Off Track',
     'Di luar jalur (tidak di dalam zona peta) dan berhenti (speed 0) selama durasi', true, false, false, 3, NULL, NULL, 1),
    ('overspeed', 'OVS', 'Overspeed',
     'Speed di atas batas selama durasi', true, true, false, 1, 50, NULL, 2),
    ('underspeed', 'UVS', 'Underspeed',
     'Bergerak (speed > 0) tetapi di bawah batas selama durasi', true, true, false, 2, 10, NULL, 3),
    ('fuel_decrease_engine_on', 'FUD', 'Fuel Decrease Engine On',
     'Mesin hidup & unit berhenti, fuel turun kumulatif sebesar ambang atau lebih selama durasi', true, false, true, 2, NULL, 5, 4),
    ('fuel_decrease_engine_off', 'FUO', 'Fuel Decrease Engine Off',
     'Fuel turun sebesar ambang atau lebih antara mesin mati dan hidup kembali', false, false, true, NULL, NULL, 5, 5);

-- =====================================================================
-- 3. Aturan alert per project (opsional per zona) — kolom kosong = ikut lapis di atasnya
-- =====================================================================
CREATE TABLE core.alert_rules (
    id                  uuid           NOT NULL DEFAULT gen_random_uuid(),
    project_id          uuid           NOT NULL,
    alert_type          varchar(40)    NOT NULL,
    zone_name           varchar(100),              -- nama area peta (lama: segment); NULL = seluruh project
    duration_minutes    numeric(6,2),
    speed_limit_kph     numeric(6,2),
    fuel_drop_liters    numeric(8,2),              -- lama: fuel_threshold −5 → 5
    description         varchar(255),
    is_active           boolean        NOT NULL DEFAULT true,
    created_at          timestamptz    NOT NULL DEFAULT now(),
    created_by          uuid,
    updated_at          timestamptz    NOT NULL DEFAULT now(),
    updated_by          uuid,

    CONSTRAINT pk_alert_rules          PRIMARY KEY (id),
    CONSTRAINT fk_alert_rules_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT fk_alert_rules_type     FOREIGN KEY (alert_type) REFERENCES core.alert_types (code),
    CONSTRAINT ck_alert_rules_zone     CHECK (zone_name IS NULL OR (zone_name <> '' AND zone_name = btrim(zone_name))),
    CONSTRAINT ck_alert_rules_values   CHECK (coalesce(duration_minutes, 0) >= 0 AND coalesce(speed_limit_kph, 0) >= 0
                                              AND coalesce(fuel_drop_liters, 0) >= 0)
);
-- satu aturan per project + jenis + zona (nama zona tidak peka huruf besar)
CREATE UNIQUE INDEX uq_alert_rules_scope ON core.alert_rules (project_id, alert_type, lower(coalesce(zone_name, '')));

CREATE TABLE core.alert_rule_recipients (
    alert_rule_id  uuid         NOT NULL,
    user_id        uuid         NOT NULL,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    created_by     uuid,

    CONSTRAINT pk_alert_rule_recipients       PRIMARY KEY (alert_rule_id, user_id),
    CONSTRAINT fk_alert_rule_recipients_rule  FOREIGN KEY (alert_rule_id) REFERENCES core.alert_rules (id) ON DELETE CASCADE,
    CONSTRAINT fk_alert_rule_recipients_user  FOREIGN KEY (user_id)       REFERENCES auth.users (id)
);
CREATE INDEX idx_alert_rule_recipients_user ON core.alert_rule_recipients (user_id);

-- =====================================================================
-- 4. Ambang efektif per jenis untuk satu project + zona:
--    bawaan jenis → aturan project (zona NULL) → aturan zona, per kolom (kosong = turun).
--    Aturan nonaktif diabaikan (seperti lama). Penerima = gabungan penerima aturan project + zona
--    (user aktif & masih anggota project). Dipakai endpoint /alert-rules/effective dan IoT (Fase C).
-- =====================================================================
CREATE FUNCTION core.effective_alert_rules(p_project uuid, p_zone_name text DEFAULT NULL)
RETURNS TABLE (alert_type varchar, duration_minutes numeric, speed_limit_kph numeric, fuel_drop_liters numeric,
               recipient_user_ids uuid[])
LANGUAGE sql STABLE AS $$
    WITH r AS (
        SELECT ar.*, (ar.zone_name IS NOT NULL) AS is_zone
          FROM core.alert_rules ar
         WHERE ar.project_id = p_project AND ar.is_active
           AND (ar.zone_name IS NULL OR lower(ar.zone_name) = lower(btrim(coalesce(p_zone_name, ''))))
    )
    SELECT t.code,
           CASE WHEN t.uses_duration    THEN coalesce(z.duration_minutes, p.duration_minutes, t.default_duration_minutes) END,
           CASE WHEN t.uses_speed_limit THEN coalesce(z.speed_limit_kph,  p.speed_limit_kph,  t.default_speed_limit_kph)  END,
           CASE WHEN t.uses_fuel_drop   THEN coalesce(z.fuel_drop_liters, p.fuel_drop_liters, t.default_fuel_drop_liters) END,
           coalesce((SELECT array_agg(DISTINCT x.user_id ORDER BY x.user_id)
                       FROM core.alert_rule_recipients x
                       JOIN auth.users u ON u.id = x.user_id AND u.is_active
                       JOIN auth.project_members m ON m.user_id = x.user_id AND m.project_id = p_project
                      WHERE x.alert_rule_id IN (p.id, z.id)), '{}')
      FROM core.alert_types t
      LEFT JOIN r p ON p.alert_type = t.code AND NOT p.is_zone
      LEFT JOIN r z ON z.alert_type = t.code AND z.is_zone
     ORDER BY t.sort_order
$$;

CREATE TRIGGER trg_fuel_calibrations_updated_at BEFORE UPDATE ON core.fuel_calibrations FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_alert_types_updated_at       BEFORE UPDATE ON core.alert_types       FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_alert_rules_updated_at       BEFORE UPDATE ON core.alert_rules       FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.fuel_calibrations, core.fuel_calibration_points TO core_api;
GRANT SELECT, UPDATE                 ON core.alert_types TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.alert_rules, core.alert_rule_recipients TO core_api;
GRANT EXECUTE ON FUNCTION core.effective_alert_rules(uuid, text) TO core_api;
