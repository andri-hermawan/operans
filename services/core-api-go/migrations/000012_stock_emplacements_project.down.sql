-- 000012 (DOWN): Stock Emplacement kembali global. Gagal bila project berbeda memuat partner & tanggal yang sama (hapus/gabung dulu).
ALTER TABLE core.stock_emplacements DROP CONSTRAINT uq_stock_emplacements_key;
ALTER TABLE core.stock_emplacements ADD CONSTRAINT uq_stock_emplacements_key UNIQUE (partner, operational_date);
ALTER TABLE core.stock_emplacements DROP CONSTRAINT fk_stock_emplacements_project;
ALTER TABLE core.stock_emplacements DROP COLUMN project_id;
