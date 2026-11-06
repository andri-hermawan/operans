-- 000002_equipment (UP)
-- Pengganti fms_lite: equipments → core.equipment, equipment_activation → core.equipment_activations.
-- ADR 0004: kode bisnis huruf besar, status baku huruf kecil, relasi lewat ID (bukan teks equipment_code).

-- =====================================================================
-- 1. core.equipment (lama: equipments). UUID diimpor SAMA dengan fms_lite (B8).
-- =====================================================================
CREATE TABLE core.equipment (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id       uuid          NOT NULL,
    equipment_code   varchar(50)   NOT NULL,      -- mis. DT10300 (huruf besar)
    equipment_alias  varchar(150),
    equipment_type   varchar(50),                 -- lama: type (mis. Dump Truck)
    brand            varchar(50),                 -- Hino
    model            varchar(50),                 -- FM260JD
    class            varchar(50),                 -- 28 Ton
    is_active        boolean       NOT NULL DEFAULT true,   -- lama: status 'active'
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,

    CONSTRAINT pk_equipment               PRIMARY KEY (id),
    CONSTRAINT uq_equipment_project_code  UNIQUE (project_id, equipment_code),
    CONSTRAINT fk_equipment_project       FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT ck_equipment_code          CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code)))
);

-- =====================================================================
-- 2. core.equipment_activations (lama: equipment_activation)
--    Status operasi harian per equipment: satu baris per equipment per tanggal operasional.
-- =====================================================================
CREATE TABLE core.equipment_activations (
    id               bigint        GENERATED ALWAYS AS IDENTITY,
    equipment_id     uuid          NOT NULL,      -- lama: equipment_code (teks)
    activation_date  date          NOT NULL,      -- lama: date_at (tanggal operasional, ADR 0004)
    status           varchar(20)   NOT NULL,      -- lama: Active / Non Active
    description      varchar(255),
    created_at       timestamptz   NOT NULL DEFAULT now(),
    created_by       uuid,
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    updated_by       uuid,

    CONSTRAINT pk_equipment_activations            PRIMARY KEY (id),
    CONSTRAINT uq_equipment_activations_eq_date    UNIQUE (equipment_id, activation_date),
    CONSTRAINT fk_equipment_activations_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_equipment_activations_status     CHECK (status IN ('active', 'inactive'))
);
CREATE INDEX idx_equipment_activations_date ON core.equipment_activations (activation_date);

CREATE TRIGGER trg_equipment_updated_at             BEFORE UPDATE ON core.equipment             FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_equipment_activations_updated_at BEFORE UPDATE ON core.equipment_activations FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

-- Equipment tidak dihapus (cukup is_active = false); aktivasi harian boleh dihapus (seperti sistem lama).
GRANT SELECT, INSERT, UPDATE         ON core.equipment             TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON core.equipment_activations TO core_api;
