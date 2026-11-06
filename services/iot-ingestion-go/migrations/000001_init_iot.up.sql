-- 000001_init_iot (UP)
-- Prasyarat, dibuat admin: database simha, schema iot (Step 4), extension postgis (Step 5a).

-- =====================================================================
-- Tabel 1: iot.devices  — daftar alat GPS yang boleh mengirim data
-- =====================================================================
CREATE TABLE iot.devices (
    id              uuid         NOT NULL DEFAULT gen_random_uuid(),
    imei            varchar(20)  NOT NULL,
    name            varchar(100),
    equipment_id    uuid,                         -- ID equipment di fms_lite (tanpa FK: beda database)
    model           varchar(50),                  -- FMC125 / FMC150 / simulator
    fuel_source     varchar(20),                  -- escort_ble / ecu / NULL
    sim_number      varchar(20),
    provider_name   varchar(100),                 -- provider kartu SIM
    api_key_hash    varchar(64),                  -- SHA-256 hex; hanya untuk alat via HTTP
    is_active       boolean      NOT NULL DEFAULT true,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT pk_devices             PRIMARY KEY (id),
    CONSTRAINT uq_devices_imei        UNIQUE (imei),
    CONSTRAINT ck_devices_imei        CHECK (imei ~ '^[0-9]{15}$'),
    CONSTRAINT ck_devices_fuel_source CHECK (fuel_source IN ('escort_ble', 'ecu'))
);

-- =====================================================================
-- Tabel 2: iot.telemetry_events — satu baris = satu kiriman data dari alat
-- =====================================================================
CREATE TABLE iot.telemetry_events (
    -- 1. Identitas & waktu
    id                     bigint       GENERATED ALWAYS AS IDENTITY,
    event_id               varchar(100) NOT NULL,          -- kunci anti-duplikat dari adapter
    device_id              uuid         NOT NULL,
    equipment_id           uuid,                           -- equipment SAAT kejadian
    event_time             timestamptz  NOT NULL,          -- waktu di alat (UTC)

    -- 2. Posisi GNSS
    location               public.geometry(Point, 4326),
    altitude_m             integer,
    heading_deg            smallint,
    speed_kph              numeric(10,2),
    satellite_count        smallint,
    hdop                   numeric(8,2),
    pdop                   numeric(8,2),
    gnss_status            smallint,

    -- 3. Mesin & jarak
    is_engine_on           boolean,
    is_moving              boolean,
    mileage_m              numeric(12,2),
    odometer_m             numeric(15,2),

    -- 4. Fuel
    fuel_level_raw         numeric(10,2),
    fuel_volume_liter      numeric(10,2),
    fuel_percent           numeric(10,2),
    fuel_delta_liter       numeric(10,2),
    fuel_temperature_c     numeric(10,2),

    -- 5. Kelistrikan & modem
    external_voltage_v     numeric(10,2),
    battery_voltage_v      numeric(10,2),
    battery_current_a      numeric(10,2),
    gsm_signal_level       smallint,
    gsm_operator_code      integer,
    sleep_mode             smallint,

    -- 6. Sensor tambahan
    accel_x_mg             numeric(10,2),
    accel_y_mg             numeric(10,2),
    accel_z_mg             numeric(10,2),
    analog_input_1_v       numeric(10,2),

    -- 7. Konteks lokasi (diisi proses olahan, bukan oleh ingestion)
    is_inside_geofence     boolean,
    geofence_fid           integer,
    location_category      varchar(100),
    segment_name           varchar(100),

    -- 8. Konteks operasional (diisi proses olahan / input manusia)
    equipment_status       varchar(20),
    vessel_load_ton        numeric(10,2),
    vessel_status          varchar(50),
    shift                  varchar(100),
    operator_name          varchar(100),
    is_breakdown           boolean,
    breakdown_description  varchar(255),

    -- 9. Data vendor yang tidak punya kolom sendiri
    metadata               jsonb        NOT NULL DEFAULT '{}'::jsonb,

    -- 10. Audit
    created_at             timestamptz  NOT NULL DEFAULT now(),   -- waktu diterima server
    created_by             uuid,
    updated_at             timestamptz,
    updated_by             uuid,

    CONSTRAINT pk_telemetry_events         PRIMARY KEY (id),
    CONSTRAINT uq_telemetry_device_event   UNIQUE (device_id, event_id),
    CONSTRAINT fk_telemetry_device         FOREIGN KEY (device_id) REFERENCES iot.devices (id),
    CONSTRAINT ck_telemetry_vessel_status  CHECK (vessel_status IN ('Loaded', 'Empty', 'Unknown')),
    CONSTRAINT ck_telemetry_equip_status   CHECK (equipment_status IN ('Running', 'Idle', 'Stop'))
);

-- Index untuk query yang paling sering: "data equipment X / alat Y, terbaru dulu"
CREATE INDEX idx_telemetry_equipment_time ON iot.telemetry_events (equipment_id, event_time DESC);
CREATE INDEX idx_telemetry_device_time    ON iot.telemetry_events (device_id, event_time DESC);

-- Keterangan kolom (terlihat di DBeaver/pgAdmin)
COMMENT ON COLUMN iot.telemetry_events.fuel_level_raw   IS 'Nilai MENTAH sensor fuel (Escort TD-150). BUKAN liter.';
COMMENT ON COLUMN iot.telemetry_events.fuel_delta_liter IS 'Pengurangan fuel (liter), hasil rumus.';
COMMENT ON COLUMN iot.telemetry_events.event_id         IS 'Kunci idempotency dari adapter; unik per device.';
COMMENT ON COLUMN iot.telemetry_events.created_at       IS 'Waktu diterima server (bandingkan dengan event_time untuk deteksi data terlambat).';
COMMENT ON COLUMN iot.devices.api_key_hash              IS 'SHA-256 hex dari API key; key asli tidak pernah disimpan.';

-- Hak akses minimum untuk akun service Go
GRANT SELECT         ON iot.devices          TO iot_ingest;
GRANT SELECT, INSERT ON iot.telemetry_events TO iot_ingest;