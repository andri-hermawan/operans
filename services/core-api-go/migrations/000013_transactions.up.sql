-- Empat input transaksi dari sistem lama (menu Record / Upload → Transaction), semuanya per project + tanggal operasional + shift:
-- Time Sheet, Ritase, WH & Notification, Rain & Slippery. Rancangan: ADR 0015. Pola sama dengan weighbridge/breakdown (ADR 0007).
--   Kolom Status lama dibuang; User/CreateDate/UpdateDate digantikan created_*/updated_*.
--   Kunci "data sama ditimpa" memakai UNIQUE NULLS NOT DISTINCT (kolom opsional yang kosong dianggap sama).

-- =====================================================================
-- 1. core.time_sheets (lama: Time Sheet) — HM awal/akhir per unit per shift
-- =====================================================================
CREATE TABLE core.time_sheets (
    id                bigint         GENERATED ALWAYS AS IDENTITY,
    project_id        uuid           NOT NULL,
    operational_date  date           NOT NULL,
    shift_id          uuid           NOT NULL,
    equipment_code    varchar(50)    NOT NULL,   -- lama: CnUnit
    equipment_id      uuid,
    operator_nik      varchar(30),               -- lama: NikOperator
    operator_name     varchar(100),
    hm_start          numeric(12,2),             -- lama: HmStart (hour meter)
    hm_end            numeric(12,2),
    hm_total          numeric(12,2),             -- lama: Total; disimpan apa adanya, kosong → hm_end − hm_start
    remark            varchar(255),
    created_at        timestamptz    NOT NULL DEFAULT now(),
    created_by        uuid,
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    updated_by        uuid,

    CONSTRAINT pk_time_sheets            PRIMARY KEY (id),
    CONSTRAINT uq_time_sheets_key        UNIQUE NULLS NOT DISTINCT (project_id, operational_date, shift_id, equipment_code),
    CONSTRAINT fk_time_sheets_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_time_sheets_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_time_sheets_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_time_sheets_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code))),
    CONSTRAINT ck_time_sheets_hm         CHECK ((hm_start IS NULL OR hm_start >= 0) AND (hm_end IS NULL OR hm_end >= 0) AND (hm_total IS NULL OR hm_total >= 0)),
    CONSTRAINT ck_time_sheets_range      CHECK (hm_start IS NULL OR hm_end IS NULL OR hm_end >= hm_start)
);
CREATE INDEX idx_time_sheets_date ON core.time_sheets (project_id, operational_date);

-- =====================================================================
-- 2. core.ritases (lama: Ritase) — produksi per jam per pasangan loader & hauler
-- =====================================================================
CREATE TABLE core.ritases (
    id                    bigint         GENERATED ALWAYS AS IDENTITY,
    project_id            uuid           NOT NULL,
    operational_date      date           NOT NULL,
    shift_id              uuid           NOT NULL,
    time_slot             varchar(20)    NOT NULL,   -- lama: Time (mis. 07.00-08.00)
    loader_code           varchar(50)    NOT NULL,   -- lama: Loader
    loader_equipment_id   uuid,
    loader_model          varchar(50),
    loader_operator_nik   varchar(30),               -- lama: NikOperatorLoader
    loader_operator_name  varchar(100),              -- lama: OperatorLoader
    loader_partner        varchar(50),               -- lama: UserLoader (berisi kode mitra, mis. TBBE)
    hauler_code           varchar(50)    NOT NULL,   -- lama: Hauler
    hauler_equipment_id   uuid,
    hauler_model          varchar(50),
    hauler_operator_nik   varchar(30),
    hauler_operator_name  varchar(100),
    hauler_partner        varchar(50),               -- lama: UserHauler
    block                 varchar(50),               -- lama: Blok
    location              varchar(100),
    dumping               varchar(100),
    material              varchar(50),
    sub_material          varchar(50),
    seam                  varchar(50),
    phase                 varchar(20),
    distance              numeric(12,2),
    ritase                numeric(12,2),             -- jumlah rit
    production            numeric(14,2),             -- lama: Prod
    production_adjusted   numeric(14,2),             -- lama: ProdAdj
    distance_volume       numeric(18,2),             -- lama: DistxVol
    productivity_factor   numeric(12,4),
    created_at            timestamptz    NOT NULL DEFAULT now(),
    created_by            uuid,
    updated_at            timestamptz    NOT NULL DEFAULT now(),
    updated_by            uuid,

    CONSTRAINT pk_ritases                 PRIMARY KEY (id),
    CONSTRAINT uq_ritases_key             UNIQUE NULLS NOT DISTINCT
                                          (project_id, operational_date, shift_id, time_slot, loader_code, hauler_code,
                                           block, location, dumping, material, sub_material, seam, phase),
    CONSTRAINT fk_ritases_project         FOREIGN KEY (project_id)          REFERENCES core.projects (id),
    CONSTRAINT fk_ritases_shift           FOREIGN KEY (shift_id)            REFERENCES core.shifts (id),
    CONSTRAINT fk_ritases_loader          FOREIGN KEY (loader_equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT fk_ritases_hauler          FOREIGN KEY (hauler_equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_ritases_codes           CHECK (loader_code <> '' AND loader_code = upper(btrim(loader_code))
                                             AND hauler_code <> '' AND hauler_code = upper(btrim(hauler_code))),
    CONSTRAINT ck_ritases_amounts         CHECK ((distance IS NULL OR distance >= 0) AND (ritase IS NULL OR ritase >= 0)
                                             AND (production IS NULL OR production >= 0) AND (production_adjusted IS NULL OR production_adjusted >= 0)
                                             AND (distance_volume IS NULL OR distance_volume >= 0) AND (productivity_factor IS NULL OR productivity_factor >= 0))
);
CREATE INDEX idx_ritases_date ON core.ritases (project_id, operational_date);

-- =====================================================================
-- 3. core.work_hour_notifications (lama: WH & Notification) — kegiatan/standby unit per jam
-- =====================================================================
CREATE TABLE core.work_hour_notifications (
    id                bigint         GENERATED ALWAYS AS IDENTITY,
    project_id        uuid           NOT NULL,
    operational_date  date           NOT NULL,
    shift_id          uuid           NOT NULL,
    equipment_code    varchar(50)    NOT NULL,   -- lama: CnUnit
    equipment_id      uuid,
    category          varchar(100)   NOT NULL,   -- mis. Standby
    sub_category      varchar(100),              -- mis. P5M
    started_at        timestamptz    NOT NULL,   -- lama: Start (HH:MM tanpa tanggal)
    ended_at          timestamptz,               -- lama: Stop
    total_hours       numeric(10,2),             -- lama: TotalHours; dihitung dari jam bila kosong
    dedicated         varchar(100),              -- mis. Waste Removal
    remark            varchar(255),
    created_at        timestamptz    NOT NULL DEFAULT now(),
    created_by        uuid,
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    updated_by        uuid,

    CONSTRAINT pk_work_hour_notifications            PRIMARY KEY (id),
    CONSTRAINT uq_work_hour_notifications_key        UNIQUE NULLS NOT DISTINCT
                                                     (project_id, operational_date, shift_id, equipment_code, category, started_at),
    CONSTRAINT fk_work_hour_notifications_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_work_hour_notifications_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_work_hour_notifications_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_work_hour_notifications_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code))),
    CONSTRAINT ck_work_hour_notifications_period     CHECK (ended_at IS NULL OR ended_at >= started_at),
    CONSTRAINT ck_work_hour_notifications_hours      CHECK (total_hours IS NULL OR total_hours >= 0)
);
CREATE INDEX idx_work_hour_notifications_date ON core.work_hour_notifications (project_id, operational_date);

-- =====================================================================
-- 4. core.rain_slippery_logs (lama: Rain & Slippery) — hujan & jalan licin per lokasi per shift
-- =====================================================================
CREATE TABLE core.rain_slippery_logs (
    id                       bigint         GENERATED ALWAYS AS IDENTITY,
    project_id               uuid           NOT NULL,
    operational_date         date           NOT NULL,
    shift_id                 uuid           NOT NULL,
    location                 varchar(100),
    rain_start_at            timestamptz,               -- lama: StartRain
    rain_stop_at             timestamptz,               -- lama: StopRain
    rain_hours               numeric(8,2),              -- lama: TotalRain; dihitung dari jam bila kosong
    wait_slippery_start_at   timestamptz,               -- lama: StartWaitSlippery
    wait_slippery_stop_at    timestamptz,
    wait_slippery_hours      numeric(8,2),
    slippery_start_at        timestamptz,               -- lama: StartSlippery
    slippery_stop_at         timestamptz,
    slippery_hours           numeric(8,2),
    rainfall_mm              numeric(8,2),              -- lama: RainFall (curah hujan, mm)
    created_at               timestamptz    NOT NULL DEFAULT now(),
    created_by               uuid,
    updated_at               timestamptz    NOT NULL DEFAULT now(),
    updated_by               uuid,

    CONSTRAINT pk_rain_slippery_logs          PRIMARY KEY (id),
    CONSTRAINT uq_rain_slippery_logs_key      UNIQUE NULLS NOT DISTINCT (project_id, operational_date, shift_id, location),
    CONSTRAINT fk_rain_slippery_logs_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT fk_rain_slippery_logs_shift    FOREIGN KEY (shift_id)   REFERENCES core.shifts (id),
    CONSTRAINT ck_rain_slippery_logs_periods  CHECK ((rain_stop_at IS NULL OR rain_start_at IS NULL OR rain_stop_at >= rain_start_at)
                                                 AND (wait_slippery_stop_at IS NULL OR wait_slippery_start_at IS NULL OR wait_slippery_stop_at >= wait_slippery_start_at)
                                                 AND (slippery_stop_at IS NULL OR slippery_start_at IS NULL OR slippery_stop_at >= slippery_start_at)),
    CONSTRAINT ck_rain_slippery_logs_amounts  CHECK ((rain_hours IS NULL OR rain_hours >= 0) AND (wait_slippery_hours IS NULL OR wait_slippery_hours >= 0)
                                                 AND (slippery_hours IS NULL OR slippery_hours >= 0) AND (rainfall_mm IS NULL OR rainfall_mm >= 0))
);
CREATE INDEX idx_rain_slippery_logs_date ON core.rain_slippery_logs (project_id, operational_date);

CREATE TRIGGER trg_time_sheets_updated_at             BEFORE UPDATE ON core.time_sheets             FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_ritases_updated_at                 BEFORE UPDATE ON core.ritases                 FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_work_hour_notifications_updated_at BEFORE UPDATE ON core.work_hour_notifications FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_rain_slippery_logs_updated_at      BEFORE UPDATE ON core.rain_slippery_logs      FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.time_sheets, core.ritases, core.work_hour_notifications, core.rain_slippery_logs TO core_api;
