-- Stock Emplacement per project: tabel ini semula global (tanpa project), kini diperlakukan sama dengan input lain
-- (weighbridge, Plan Yearly/Monthly): setiap baris milik satu project; kunci "data sama ditimpa" = project + partner + tanggal.
-- Pengaman: bila tabel sudah berisi data, migration berhenti. Isi project_id baris lama secara sadar (jangan ditebak), lalu ulangi.
ALTER TABLE core.stock_emplacements ADD COLUMN project_id uuid;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM core.stock_emplacements WHERE project_id IS NULL) THEN
        RAISE EXCEPTION 'core.stock_emplacements sudah berisi data: isi kolom project_id baris lama dengan project yang benar (UPDATE), lalu jalankan migration ini lagi';
    END IF;
END $$;

ALTER TABLE core.stock_emplacements ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE core.stock_emplacements ADD CONSTRAINT fk_stock_emplacements_project
    FOREIGN KEY (project_id) REFERENCES core.projects (id);

ALTER TABLE core.stock_emplacements DROP CONSTRAINT uq_stock_emplacements_key;
ALTER TABLE core.stock_emplacements ADD CONSTRAINT uq_stock_emplacements_key UNIQUE (project_id, partner, operational_date);
