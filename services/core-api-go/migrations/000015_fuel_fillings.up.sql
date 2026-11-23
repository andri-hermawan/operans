-- Pengisian BBM unit (menu Record / Upload → Transaction → Fuel Filling): satu baris per pengisian per unit per shift.
-- Pola sama dengan Time Sheet (ADR 0015): project + tanggal operasional + shift, kolom audit, trigger updated_at.
--   Lama: Date → operational_date, Shift → shift_id, CnUnit → equipment_code, Model → equipment_model,
--         HmPengisianSebelumnya → hm_previous, HmPengisian → hm_filling, TotalHours → total_hours,
--         FuelUsageLtr → fuel_usage_liter, FuelConsLtrHrs → fuel_consumption_lph, WasteRemoval … FinalSupport → *_liter.
--   Dibuang: id lama, ProjectCode (project dari konteks login), Status, User, CreateDate, UpdateDate.
--   Kunci "data sama ditimpa": project + tanggal + shift + unit + HM pengisian (satu unit bisa isi lebih dari sekali per shift).

CREATE TABLE core.fuel_fillings (
    id                      bigint         GENERATED ALWAYS AS IDENTITY,
    project_id              uuid           NOT NULL,
    operational_date        date           NOT NULL,
    shift_id                uuid           NOT NULL,
    equipment_code          varchar(50)    NOT NULL,   -- lama: CnUnit
    equipment_id            uuid,
    equipment_model         varchar(50),               -- lama: Model (mis. D8T)
    hm_previous             numeric(12,2),             -- lama: HmPengisianSebelumnya
    hm_filling              numeric(12,2),             -- lama: HmPengisian
    total_hours             numeric(12,2),             -- lama: TotalHours; disimpan apa adanya, kosong → hm_filling − hm_previous
    fuel_usage_liter        numeric(12,2),             -- lama: FuelUsageLtr
    fuel_consumption_lph    numeric(12,2),             -- lama: FuelConsLtrHrs; kosong → fuel_usage_liter / total_hours
    waste_removal_liter     numeric(12,2),             -- pemakaian per aktivitas (liter)
    coal_mining_liter       numeric(12,2),
    hauling_liter           numeric(12,2),
    rental_liter            numeric(12,2),
    dewatering_liter        numeric(12,2),
    final_support_liter     numeric(12,2),
    created_at              timestamptz    NOT NULL DEFAULT now(),
    created_by              uuid,
    updated_at              timestamptz    NOT NULL DEFAULT now(),
    updated_by              uuid,

    CONSTRAINT pk_fuel_fillings            PRIMARY KEY (id),
    CONSTRAINT uq_fuel_fillings_key        UNIQUE NULLS NOT DISTINCT (project_id, operational_date, shift_id, equipment_code, hm_filling),
    CONSTRAINT fk_fuel_fillings_project    FOREIGN KEY (project_id)   REFERENCES core.projects (id),
    CONSTRAINT fk_fuel_fillings_shift      FOREIGN KEY (shift_id)     REFERENCES core.shifts (id),
    CONSTRAINT fk_fuel_fillings_equipment  FOREIGN KEY (equipment_id) REFERENCES core.equipment (id),
    CONSTRAINT ck_fuel_fillings_code       CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code))),
    CONSTRAINT ck_fuel_fillings_amounts    CHECK (
        (hm_previous IS NULL OR hm_previous >= 0) AND (hm_filling IS NULL OR hm_filling >= 0)
        AND (total_hours IS NULL OR total_hours >= 0) AND (fuel_usage_liter IS NULL OR fuel_usage_liter >= 0)
        AND (fuel_consumption_lph IS NULL OR fuel_consumption_lph >= 0)
        AND (waste_removal_liter IS NULL OR waste_removal_liter >= 0) AND (coal_mining_liter IS NULL OR coal_mining_liter >= 0)
        AND (hauling_liter IS NULL OR hauling_liter >= 0) AND (rental_liter IS NULL OR rental_liter >= 0)
        AND (dewatering_liter IS NULL OR dewatering_liter >= 0) AND (final_support_liter IS NULL OR final_support_liter >= 0)),
    CONSTRAINT ck_fuel_fillings_range      CHECK (hm_previous IS NULL OR hm_filling IS NULL OR hm_filling >= hm_previous)
);
CREATE INDEX idx_fuel_fillings_date ON core.fuel_fillings (project_id, operational_date);

CREATE TRIGGER trg_fuel_fillings_updated_at BEFORE UPDATE ON core.fuel_fillings FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.fuel_fillings TO core_api;
