-- Kembali ke numeric(12,4); nilai yang bukan angka akan membuat perintah ini gagal (hapus/perbaiki dulu).
ALTER TABLE core.ritases DROP CONSTRAINT ck_ritases_amounts;
ALTER TABLE core.ritases ALTER COLUMN productivity_factor TYPE numeric(12,4) USING NULLIF(btrim(productivity_factor), '')::numeric;
ALTER TABLE core.ritases ADD CONSTRAINT ck_ritases_amounts
    CHECK ((distance IS NULL OR distance >= 0) AND (ritase IS NULL OR ritase >= 0)
       AND (production IS NULL OR production >= 0) AND (production_adjusted IS NULL OR production_adjusted >= 0)
       AND (distance_volume IS NULL OR distance_volume >= 0) AND (productivity_factor IS NULL OR productivity_factor >= 0));
