-- Kembali ke bentuk lama: baris untuk unit yang tidak ada di master (equipment_id kosong) dihapus.
DELETE FROM core.equipment_activations WHERE equipment_id IS NULL;

DROP INDEX core.idx_equipment_activations_date;
CREATE INDEX idx_equipment_activations_date ON core.equipment_activations (activation_date);

ALTER TABLE core.equipment_activations DROP CONSTRAINT uq_equipment_activations_key;
ALTER TABLE core.equipment_activations ADD CONSTRAINT uq_equipment_activations_eq_date UNIQUE (equipment_id, activation_date);

ALTER TABLE core.equipment_activations DROP CONSTRAINT ck_equipment_activations_code;
ALTER TABLE core.equipment_activations DROP CONSTRAINT fk_equipment_activations_project;
ALTER TABLE core.equipment_activations ALTER COLUMN equipment_id SET NOT NULL;
ALTER TABLE core.equipment_activations DROP COLUMN equipment_code;
ALTER TABLE core.equipment_activations DROP COLUMN project_id;
