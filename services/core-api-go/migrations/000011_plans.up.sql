-- Rencana produksi per project: Plan Yearly (target per bulan dalam satu tahun) dan Plan Monthly (target per tanggal
-- dalam satu bulan). Lama: tabel Plan Yearly / Plan Month di SQL Server. Rancangan: ADR 0014.
--   Mitra → partner, Year → plan_year, Month → plan_month, Name → plan_name, T01..T31 → day_01..day_31.
--   Kolom total disimpan apa adanya dari input (pada data lama total bisa beda tipis dari jumlah akibat pembulatan);
--   bila total tidak diisi, dihitung dari jumlah bulan/hari oleh service.
--   Kolom Status lama dibuang; User/CreateDate/UpdateDate digantikan created_*/updated_*.
--   Angka kosong disimpan 0 (satuan mengikuti Description/Uom: BCM, ton, dst).

-- =====================================================================
-- 1. core.yearly_plans — satu baris per mitra, tahun, lokasi, deskripsi, dan nama
-- =====================================================================
CREATE TABLE core.yearly_plans (
    id               bigint GENERATED ALWAYS AS IDENTITY,
    project_id       uuid          NOT NULL,
    partner          varchar(50)   NOT NULL,
    plan_year        smallint      NOT NULL,
    location         varchar(100)  NOT NULL,
    description      varchar(100)  NOT NULL,
    plan_name        varchar(100)  NOT NULL,
    total            numeric(16,2) NOT NULL DEFAULT 0,
    jan              numeric(16,2) NOT NULL DEFAULT 0,
    feb              numeric(16,2) NOT NULL DEFAULT 0,
    mar              numeric(16,2) NOT NULL DEFAULT 0,
    apr              numeric(16,2) NOT NULL DEFAULT 0,
    may              numeric(16,2) NOT NULL DEFAULT 0,
    jun              numeric(16,2) NOT NULL DEFAULT 0,
    jul              numeric(16,2) NOT NULL DEFAULT 0,
    aug              numeric(16,2) NOT NULL DEFAULT 0,
    sep              numeric(16,2) NOT NULL DEFAULT 0,
    oct              numeric(16,2) NOT NULL DEFAULT 0,
    nov              numeric(16,2) NOT NULL DEFAULT 0,
    dec              numeric(16,2) NOT NULL DEFAULT 0,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    CONSTRAINT pk_yearly_plans         PRIMARY KEY (id),
    CONSTRAINT fk_yearly_plans_project FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT uq_yearly_plans_key     UNIQUE (project_id, partner, plan_year, location, description, plan_name),
    CONSTRAINT ck_yearly_plans_year    CHECK (plan_year BETWEEN 2000 AND 2100),
    CONSTRAINT ck_yearly_plans_amounts CHECK (total >= 0 AND jan >= 0 AND feb >= 0 AND mar >= 0 AND apr >= 0 AND may >= 0 AND jun >= 0 AND jul >= 0 AND aug >= 0 AND sep >= 0 AND oct >= 0 AND nov >= 0 AND dec >= 0)
);

CREATE TRIGGER trg_yearly_plans_updated_at BEFORE UPDATE ON core.yearly_plans
    FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

-- =====================================================================
-- 2. core.monthly_plans — satu baris per mitra, tahun, bulan, deskripsi, dan nama
-- =====================================================================
CREATE TABLE core.monthly_plans (
    id               bigint GENERATED ALWAYS AS IDENTITY,
    project_id       uuid          NOT NULL,
    partner          varchar(50)   NOT NULL,
    plan_year        smallint      NOT NULL,
    plan_month       smallint      NOT NULL,
    description      varchar(100)  NOT NULL,
    plan_name        varchar(100)  NOT NULL,
    uom              varchar(20)   NOT NULL,
    total            numeric(16,2) NOT NULL DEFAULT 0,
    day_01           numeric(16,2) NOT NULL DEFAULT 0,
    day_02           numeric(16,2) NOT NULL DEFAULT 0,
    day_03           numeric(16,2) NOT NULL DEFAULT 0,
    day_04           numeric(16,2) NOT NULL DEFAULT 0,
    day_05           numeric(16,2) NOT NULL DEFAULT 0,
    day_06           numeric(16,2) NOT NULL DEFAULT 0,
    day_07           numeric(16,2) NOT NULL DEFAULT 0,
    day_08           numeric(16,2) NOT NULL DEFAULT 0,
    day_09           numeric(16,2) NOT NULL DEFAULT 0,
    day_10           numeric(16,2) NOT NULL DEFAULT 0,
    day_11           numeric(16,2) NOT NULL DEFAULT 0,
    day_12           numeric(16,2) NOT NULL DEFAULT 0,
    day_13           numeric(16,2) NOT NULL DEFAULT 0,
    day_14           numeric(16,2) NOT NULL DEFAULT 0,
    day_15           numeric(16,2) NOT NULL DEFAULT 0,
    day_16           numeric(16,2) NOT NULL DEFAULT 0,
    day_17           numeric(16,2) NOT NULL DEFAULT 0,
    day_18           numeric(16,2) NOT NULL DEFAULT 0,
    day_19           numeric(16,2) NOT NULL DEFAULT 0,
    day_20           numeric(16,2) NOT NULL DEFAULT 0,
    day_21           numeric(16,2) NOT NULL DEFAULT 0,
    day_22           numeric(16,2) NOT NULL DEFAULT 0,
    day_23           numeric(16,2) NOT NULL DEFAULT 0,
    day_24           numeric(16,2) NOT NULL DEFAULT 0,
    day_25           numeric(16,2) NOT NULL DEFAULT 0,
    day_26           numeric(16,2) NOT NULL DEFAULT 0,
    day_27           numeric(16,2) NOT NULL DEFAULT 0,
    day_28           numeric(16,2) NOT NULL DEFAULT 0,
    day_29           numeric(16,2) NOT NULL DEFAULT 0,
    day_30           numeric(16,2) NOT NULL DEFAULT 0,
    day_31           numeric(16,2) NOT NULL DEFAULT 0,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,
    CONSTRAINT pk_monthly_plans         PRIMARY KEY (id),
    CONSTRAINT fk_monthly_plans_project FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT uq_monthly_plans_key     UNIQUE (project_id, partner, plan_year, plan_month, description, plan_name),
    CONSTRAINT ck_monthly_plans_period  CHECK (plan_year BETWEEN 2000 AND 2100 AND plan_month BETWEEN 1 AND 12),
    CONSTRAINT ck_monthly_plans_amounts CHECK (total >= 0 AND day_01 >= 0 AND day_02 >= 0 AND day_03 >= 0 AND day_04 >= 0 AND day_05 >= 0 AND day_06 >= 0 AND day_07 >= 0 AND day_08 >= 0 AND day_09 >= 0 AND day_10 >= 0 AND day_11 >= 0 AND day_12 >= 0 AND day_13 >= 0 AND day_14 >= 0 AND day_15 >= 0 AND day_16 >= 0 AND day_17 >= 0 AND day_18 >= 0 AND day_19 >= 0 AND day_20 >= 0 AND day_21 >= 0 AND day_22 >= 0 AND day_23 >= 0 AND day_24 >= 0 AND day_25 >= 0 AND day_26 >= 0 AND day_27 >= 0 AND day_28 >= 0 AND day_29 >= 0 AND day_30 >= 0 AND day_31 >= 0)
);

CREATE TRIGGER trg_monthly_plans_updated_at BEFORE UPDATE ON core.monthly_plans
    FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.yearly_plans, core.monthly_plans TO core_api;
