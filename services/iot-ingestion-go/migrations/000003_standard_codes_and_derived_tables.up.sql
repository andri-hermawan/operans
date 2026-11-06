-- 000003_standard_codes_and_derived_tables (UP)
-- ADR 0004: kode status baku (huruf kecil) + waktu operasional (operational_date, shift_code).
-- ADR 0005: tabel turunan pengganti equipment_status, fuels, geofences, alerts dari fms_lite.
-- Kode baku:
--   equipment_status : running | idle | stop | offline
--   vessel_status    : loaded | empty | unknown
--   fuel event       : increase | decrease
--   geofence event   : enter | exit
--   alert_type       : overspeed | underspeed | off_track | fuel_decrease_engine_on | fuel_decrease_engine_off

-- =====================================================================
-- 1. telemetry_events: kode baku + waktu operasional
-- =====================================================================
ALTER TABLE iot.telemetry_events DROP CONSTRAINT ck_telemetry_vessel_status;
ALTER TABLE iot.telemetry_events DROP CONSTRAINT ck_telemetry_equip_status;
ALTER TABLE iot.telemetry_events
    ADD CONSTRAINT ck_telemetry_vessel_status    CHECK (vessel_status IN ('loaded', 'empty', 'unknown')),
    ADD CONSTRAINT ck_telemetry_equipment_status CHECK (equipment_status IN ('running', 'idle', 'stop', 'offline'));

-- Kolom teks "shift" diganti dua kolom terstruktur
ALTER TABLE iot.telemetry_events DROP COLUMN shift;
ALTER TABLE iot.telemetry_events
    ADD COLUMN operational_date date,         -- tanggal DIMULAINYA shift (jam lokal project)
    ADD COLUMN shift_code       varchar(20);  -- mis. SHIFT_1, SHIFT_2

CREATE INDEX idx_telemetry_project_opdate ON iot.telemetry_events (project_id, operational_date, shift_code);

COMMENT ON COLUMN iot.telemetry_events.operational_date IS 'Tanggal operasional = tanggal mulai shift (ADR 0004). Data 02:00 di Shift 2 (19:00-07:00) milik tanggal sebelumnya.';

-- =====================================================================
-- 2. equipment_live_states: kondisi TERAKHIR tiap equipment (lama: equipment_status)
--    Satu baris per equipment, ditimpa setiap data baru masuk. Sumber data peta live.
-- =====================================================================
CREATE TABLE iot.equipment_live_states (
    equipment_id           uuid         NOT NULL,
    project_id             uuid,
    device_id              uuid         NOT NULL,
    telemetry_event_id     bigint       NOT NULL,   -- data telemetry terakhir (lama: log_id)
    event_time             timestamptz  NOT NULL,   -- waktu data terakhir di alat
    received_at            timestamptz  NOT NULL,   -- waktu data terakhir diterima server
    operational_date       date,                    -- lama: alert_date
    shift_code             varchar(20),             -- lama: shift

    -- Posisi
    location               public.geometry(Point, 4326),
    speed_kph              numeric(10,2),
    heading_deg            smallint,

    -- Status
    is_engine_on           boolean,
    equipment_status       varchar(20),
    vessel_status          varchar(20),
    vessel_load_ton        numeric(10,2),
    gsm_signal_level       smallint,
    mileage_m              numeric(12,2),

    -- Geofence
    is_inside_geofence     boolean,
    geofence_fid           integer,
    location_category      varchar(100),
    segment_name           varchar(100),

    -- Fuel (nilai valid terakhir)
    fuel_level_raw         numeric(10,2),
    fuel_volume_liter      numeric(10,2),
    fuel_percent           numeric(10,2),
    fuel_delta_liter       numeric(10,2),
    fuel_temperature_c     numeric(10,2),

    -- Operasional
    operator_name          varchar(100),
    is_breakdown           boolean,
    breakdown_description  varchar(255),
    alert_count            integer      NOT NULL DEFAULT 0,   -- jumlah alert pada operational_date + shift_code ini

    created_at             timestamptz  NOT NULL DEFAULT now(),
    updated_at             timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT pk_equipment_live_states             PRIMARY KEY (equipment_id),
    CONSTRAINT fk_live_states_device                FOREIGN KEY (device_id) REFERENCES iot.devices (id),
    CONSTRAINT ck_live_states_equipment_status      CHECK (equipment_status IN ('running', 'idle', 'stop', 'offline')),
    CONSTRAINT ck_live_states_vessel_status         CHECK (vessel_status IN ('loaded', 'empty', 'unknown'))
);
CREATE INDEX idx_live_states_project ON iot.equipment_live_states (project_id);

-- =====================================================================
-- 3. fuel_events: kejadian fuel BERMAKNA saja (lama: fuels, satu baris per paket)
--    increase >= 10 L (isi BBM), decrease >= 5 L — ambang bawaan ADR 0005, bisa diatur per project.
-- =====================================================================
CREATE TABLE iot.fuel_events (
    id                     bigint       GENERATED ALWAYS AS IDENTITY,
    project_id             uuid,
    equipment_id           uuid         NOT NULL,
    device_id              uuid         NOT NULL,
    telemetry_event_id     bigint       NOT NULL,   -- data yang memicu kejadian (lama: log_id)
    event_type             varchar(20)  NOT NULL,
    event_time             timestamptz  NOT NULL,
    operational_date       date,
    shift_code             varchar(20),

    -- Fuel
    fuel_delta_liter       numeric(10,2) NOT NULL,  -- negatif = turun
    fuel_level_raw         numeric(10,2),
    fuel_volume_liter      numeric(10,2),
    fuel_percent           numeric(10,2),
    fuel_temperature_c     numeric(10,2),

    -- Kondisi saat kejadian
    location               public.geometry(Point, 4326),
    speed_kph              numeric(10,2),
    is_engine_on           boolean,
    equipment_status       varchar(20),
    vessel_status          varchar(20),
    vessel_load_ton        numeric(10,2),
    mileage_m              numeric(12,2),
    is_inside_geofence     boolean,
    geofence_fid           integer,
    location_category      varchar(100),
    segment_name           varchar(100),
    description            varchar(255),

    created_at             timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT pk_fuel_events         PRIMARY KEY (id),
    CONSTRAINT fk_fuel_events_device  FOREIGN KEY (device_id) REFERENCES iot.devices (id),
    CONSTRAINT ck_fuel_events_type    CHECK (event_type IN ('increase', 'decrease'))
);
CREATE INDEX idx_fuel_events_equipment_time ON iot.fuel_events (equipment_id, event_time DESC);
CREATE INDEX idx_fuel_events_project_opdate ON iot.fuel_events (project_id, operational_date, shift_code);

-- =====================================================================
-- 4. geofence_events: masuk / keluar zona (lama: geofences, IN/OUT)
-- =====================================================================
CREATE TABLE iot.geofence_events (
    id                     bigint       GENERATED ALWAYS AS IDENTITY,
    project_id             uuid,
    equipment_id           uuid         NOT NULL,
    device_id              uuid         NOT NULL,
    telemetry_event_id     bigint       NOT NULL,
    event_type             varchar(10)  NOT NULL,
    event_time             timestamptz  NOT NULL,
    operational_date       date,
    shift_code             varchar(20),

    -- Zona yang dimasuki / ditinggalkan
    geofence_fid           integer,
    location_category      varchar(100),
    segment_name           varchar(100),
    is_alert               boolean      NOT NULL DEFAULT false,

    -- Kondisi saat kejadian
    location               public.geometry(Point, 4326),
    speed_kph              numeric(10,2),
    is_engine_on           boolean,
    equipment_status       varchar(20),
    vessel_status          varchar(20),
    vessel_load_ton        numeric(10,2),
    fuel_level_raw         numeric(10,2),
    mileage_m              numeric(12,2),
    description            varchar(255),

    created_at             timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT pk_geofence_events         PRIMARY KEY (id),
    CONSTRAINT fk_geofence_events_device  FOREIGN KEY (device_id) REFERENCES iot.devices (id),
    CONSTRAINT ck_geofence_events_type    CHECK (event_type IN ('enter', 'exit'))
);
CREATE INDEX idx_geofence_events_equipment_time ON iot.geofence_events (equipment_id, event_time DESC);
CREATE INDEX idx_geofence_events_project_opdate ON iot.geofence_events (project_id, operational_date, shift_code);

-- =====================================================================
-- 5. alerts: satu baris = satu kejadian dari awal sampai selesai
--    ended_at NULL = masih berlangsung. Status: berlangsung / dibaca / diakui / selesai.
-- =====================================================================
CREATE TABLE iot.alerts (
    id                     bigint       GENERATED ALWAYS AS IDENTITY,
    project_id             uuid,
    equipment_id           uuid         NOT NULL,
    device_id              uuid         NOT NULL,
    telemetry_event_id     bigint,                  -- data yang memicu alert
    alert_type             varchar(40)  NOT NULL,
    started_at             timestamptz  NOT NULL,   -- awal kejadian (lama: created_at)
    ended_at               timestamptz,             -- akhir kejadian (lama: resolved_at)
    operational_date       date,
    shift_code             varchar(20),
    threshold_value        numeric(10,2),           -- ambang aturan saat itu (batas speed / fuel)

    -- Kondisi saat kejadian
    location               public.geometry(Point, 4326),
    speed_kph              numeric(10,2),
    is_engine_on           boolean,
    equipment_status       varchar(20),
    vessel_status          varchar(20),
    vessel_load_ton        numeric(10,2),
    mileage_m              numeric(12,2),
    fuel_level_raw         numeric(10,2),
    fuel_volume_liter      numeric(10,2),
    fuel_percent           numeric(10,2),
    fuel_delta_liter       numeric(10,2),
    fuel_temperature_c     numeric(10,2),
    is_inside_geofence     boolean,
    geofence_fid           integer,
    location_category      varchar(100),
    segment_name           varchar(100),
    operator_name          varchar(100),

    -- Tindak lanjut pengguna
    is_read                boolean      NOT NULL DEFAULT false,
    acknowledged_at        timestamptz,
    acknowledged_by        uuid,
    resolved_by            uuid,                    -- diisi bila ditutup manual oleh pengguna

    created_at             timestamptz  NOT NULL DEFAULT now(),
    updated_at             timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT pk_alerts         PRIMARY KEY (id),
    CONSTRAINT fk_alerts_device  FOREIGN KEY (device_id) REFERENCES iot.devices (id),
    CONSTRAINT ck_alerts_type    CHECK (alert_type IN ('overspeed', 'underspeed', 'off_track',
                                                       'fuel_decrease_engine_on', 'fuel_decrease_engine_off')),
    CONSTRAINT ck_alerts_period  CHECK (ended_at IS NULL OR ended_at >= started_at)
);
-- Mencegah alert ganda: hanya SATU alert yang masih berlangsung per equipment + jenis
CREATE UNIQUE INDEX uq_alerts_one_open          ON iot.alerts (equipment_id, alert_type) WHERE ended_at IS NULL;
CREATE INDEX        idx_alerts_project_started  ON iot.alerts (project_id, started_at DESC);
CREATE INDEX        idx_alerts_equipment_started ON iot.alerts (equipment_id, started_at DESC);
CREATE INDEX        idx_alerts_project_opdate   ON iot.alerts (project_id, operational_date, shift_code);

-- =====================================================================
-- Hak akses minimum untuk service Go
-- =====================================================================
GRANT SELECT, INSERT, UPDATE ON iot.equipment_live_states TO iot_ingest;  -- UPDATE: ditimpa tiap data baru
GRANT SELECT, INSERT         ON iot.fuel_events           TO iot_ingest;
GRANT SELECT, INSERT         ON iot.geofence_events       TO iot_ingest;
GRANT SELECT, INSERT, UPDATE ON iot.alerts                TO iot_ingest;  -- UPDATE: menutup / mengakui alert
