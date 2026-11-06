-- 000003_daily_inputs (UP) — ADR 0007
-- Pengganti fms_lite: daily_setting_operator → core.operator_assignments,
-- breakdown_status → core.equipment_breakdowns, weighbridge → core.weighbridge_tickets,
-- + core.reference_values (daftar pilihan yang kelak bisa ditambah admin).
-- Kode unit disimpan sebagai teks (huruf besar) + equipment_id yang ditautkan bila unit ada di master.

-- =====================================================================
-- 0. Daftar pilihan (kelompok + kode baku + label tampilan)
-- =====================================================================
CREATE TABLE core.reference_values (
    id           bigint        GENERATED ALWAYS AS IDENTITY,
    group_code   varchar(50)   NOT NULL,   -- breakdown_category | repair_status
    code         varchar(50)   NOT NULL,   -- kode baku (ADR 0004)
    label        varchar(100)  NOT NULL,   -- teks yang dipakai di Excel/tampilan
    sort_order   smallint      NOT NULL DEFAULT 0,
    is_active    boolean       NOT NULL DEFAULT true,
    created_at   timestamptz   NOT NULL DEFAULT now(),
    created_by   uuid,
    updated_at   timestamptz   NOT NULL DEFAULT now(),
    updated_by   uuid,

    CONSTRAINT pk_reference_values        PRIMARY KEY (id),
    CONSTRAINT uq_reference_values_code   UNIQUE (group_code, code),
    CONSTRAINT uq_reference_values_label  UNIQUE (group_code, label),
    CONSTRAINT ck_reference_values_code   CHECK (code ~ '^[a-z][a-z0-9_]*$')
);

INSERT INTO core.reference_values (group_code, code, label, sort_order) VALUES
    ('breakdown_category', 'unscheduled_maintenance', 'Unsch Maintenance',    1),
    ('breakdown_category', 'scheduled_maintenance',   'Sch Maintenance',      2),
    ('breakdown_category', 'incident_maintenance',    'Incident Maintenance', 3),
    ('breakdown_category', 'tyre_maintenance',        'Tyre Maintenance',     4),
    ('breakdown_category', 'operation',               'Operation',            5),
    ('breakdown_category', 'loading',                 'Loading',              6),
    ('breakdown_category', 'waiting',                 'Waiting',              7),
    ('breakdown_category', 'other',                   'Other',                8),
    ('repair_status',      'on_progress',             'On Progress',          1),
    ('repair_status',      'done',                    'Done',                 2),
    ('repair_status',      'pending',                 'Pending',              3),
    ('repair_status',      'waiting_parts',           'Waiting Parts',        4),
    ('repair_status',      'waiting_ba',              'Waiting BA',           5);

-- =====================================================================
-- 1. core.operator_assignments (lama: daily_setting_operator)
-- =====================================================================
CREATE TABLE core.operator_assignments (
    id                bigint        GENERATED ALWAYS AS IDENTITY,
    project_id        uuid          NOT NULL,
    operational_date  date          NOT NULL,   -- lama: date_at
    shift_id          uuid          NOT NULL,   -- lama: shift (teks)
    equipment_code    varchar(50)   NOT NULL,   -- lama: equipment_code / "Asset ID"
    equipment_id      uuid,                     -- ditautkan bila unit ada di master
    operator_name     varchar(100)  NOT NULL,
    description       varchar(255),
    created_at        timestamptz   NOT NULL DEFAULT now(),
    created_by        uuid,
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    updated_by        uuid,

    CONSTRAINT pk_operator_assignments            PRIMARY KEY (id),
    CONSTRAINT uq_operator_assignments_key        UNIQUE (project_id, operational_date, shift_id, equipment_code),
    CONSTRAINT fk_operator_assignments_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_operator_assignments_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_operator_assignments_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_operator_assignments_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code)))
);
CREATE INDEX idx_operator_assignments_date ON core.operator_assignments (project_id, operational_date);

-- =====================================================================
-- 2. core.equipment_breakdowns (lama: breakdown_status)
-- =====================================================================
CREATE TABLE core.equipment_breakdowns (
    id                    bigint         GENERATED ALWAYS AS IDENTITY,
    project_id            uuid           NOT NULL,
    operational_date      date           NOT NULL,
    shift_id              uuid           NOT NULL,
    equipment_code        varchar(50)    NOT NULL,
    equipment_id          uuid,
    status                varchar(20)    NOT NULL,  -- breakdown | continue | ready
    category_code         varchar(50),               -- reference_values.breakdown_category
    started_at            timestamptz,               -- lama: time_start (tanpa tanggal)
    ended_at              timestamptz,               -- lama: time_end
    duration_hours        numeric(10,2),             -- lama: duration; dihitung bila kosong
    repair_status_code    varchar(50),               -- reference_values.repair_status
    description           varchar(255),
    location              varchar(100),
    created_at            timestamptz    NOT NULL DEFAULT now(),
    created_by            uuid,
    updated_at            timestamptz    NOT NULL DEFAULT now(),
    updated_by            uuid,

    CONSTRAINT pk_equipment_breakdowns            PRIMARY KEY (id),
    CONSTRAINT uq_equipment_breakdowns_key        UNIQUE NULLS NOT DISTINCT
                                                  (project_id, operational_date, shift_id, equipment_code, category_code, started_at),
    CONSTRAINT fk_equipment_breakdowns_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_equipment_breakdowns_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_equipment_breakdowns_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_equipment_breakdowns_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code))),
    CONSTRAINT ck_equipment_breakdowns_status     CHECK (status IN ('breakdown', 'continue', 'ready')),
    CONSTRAINT ck_equipment_breakdowns_period     CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at),
    CONSTRAINT ck_equipment_breakdowns_duration   CHECK (duration_hours IS NULL OR duration_hours >= 0)
);
CREATE INDEX idx_equipment_breakdowns_date ON core.equipment_breakdowns (project_id, operational_date);

-- =====================================================================
-- 3. core.weighbridge_tickets (lama: weighbridge). Berat dalam kg.
-- =====================================================================
CREATE TABLE core.weighbridge_tickets (
    id                bigint         GENERATED ALWAYS AS IDENTITY,
    project_id        uuid           NOT NULL,
    operational_date  date           NOT NULL,
    shift_id          uuid           NOT NULL,
    ticket_no         varchar(50),
    equipment_code    varchar(50)    NOT NULL,   -- truk (bisa milik transporter lain)
    equipment_id      uuid,
    product           varchar(50),
    gross_kg          numeric(12,2),
    tare_kg           numeric(12,2),
    net_kg            numeric(12,2),             -- = gross − tare (ADR 0007)
    recipient         varchar(100),
    customer          varchar(100),
    transporter       varchar(100),
    gross_at          timestamptz,               -- lama: gross_time (tanggal 1970)
    tare_at           timestamptz,               -- lama: tare_time
    gross_operator    varchar(50),
    tare_operator     varchar(50),
    description       varchar(255),
    location          varchar(100),
    created_at        timestamptz    NOT NULL DEFAULT now(),
    created_by        uuid,
    updated_at        timestamptz    NOT NULL DEFAULT now(),
    updated_by        uuid,

    CONSTRAINT pk_weighbridge_tickets            PRIMARY KEY (id),
    CONSTRAINT uq_weighbridge_tickets_key        UNIQUE NULLS NOT DISTINCT (project_id, operational_date, ticket_no),
    CONSTRAINT fk_weighbridge_tickets_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_weighbridge_tickets_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_weighbridge_tickets_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_weighbridge_tickets_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code))),
    CONSTRAINT ck_weighbridge_tickets_weights    CHECK ((gross_kg IS NULL OR gross_kg >= 0) AND (tare_kg IS NULL OR tare_kg >= 0)),
    CONSTRAINT ck_weighbridge_tickets_net        CHECK (net_kg IS NULL OR gross_kg IS NULL OR tare_kg IS NULL OR net_kg = gross_kg - tare_kg)
);
CREATE INDEX idx_weighbridge_tickets_date ON core.weighbridge_tickets (project_id, operational_date);

CREATE TRIGGER trg_reference_values_updated_at      BEFORE UPDATE ON core.reference_values      FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_operator_assignments_updated_at  BEFORE UPDATE ON core.operator_assignments  FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_equipment_breakdowns_updated_at  BEFORE UPDATE ON core.equipment_breakdowns  FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_weighbridge_tickets_updated_at   BEFORE UPDATE ON core.weighbridge_tickets   FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE         ON core.reference_values     TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.operator_assignments TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.equipment_breakdowns TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.weighbridge_tickets  TO core_api;
