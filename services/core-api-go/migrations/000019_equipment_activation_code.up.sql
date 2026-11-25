-- Equipment Activation: tambah equipment_code (varchar 50) + project_id seperti weighbridge_tickets.
-- Unit boleh belum ada di master: equipment_id kini opsional (terisi bila kodenya ada di master equipment project itu).
-- Kunci "data sama ditimpa" = project + kode unit + tanggal. Baris lama diisi dari master equipment lewat equipment_id.
ALTER TABLE core.equipment_activations ADD COLUMN project_id uuid;
ALTER TABLE core.equipment_activations ADD COLUMN equipment_code varchar(50);

UPDATE core.equipment_activations a
   SET project_id = e.project_id, equipment_code = e.equipment_code
  FROM core.equipment e
 WHERE e.id = a.equipment_id;

ALTER TABLE core.equipment_activations ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE core.equipment_activations ALTER COLUMN equipment_code SET NOT NULL;
ALTER TABLE core.equipment_activations ALTER COLUMN equipment_id DROP NOT NULL;

ALTER TABLE core.equipment_activations ADD CONSTRAINT fk_equipment_activations_project
    FOREIGN KEY (project_id) REFERENCES core.projects (id);
ALTER TABLE core.equipment_activations ADD CONSTRAINT ck_equipment_activations_code
    CHECK (equipment_code <> '' AND equipment_code = upper(btrim(equipment_code)));

ALTER TABLE core.equipment_activations DROP CONSTRAINT uq_equipment_activations_eq_date;
ALTER TABLE core.equipment_activations ADD CONSTRAINT uq_equipment_activations_key
    UNIQUE (project_id, equipment_code, activation_date);

DROP INDEX core.idx_equipment_activations_date;
CREATE INDEX idx_equipment_activations_date ON core.equipment_activations (project_id, activation_date);
