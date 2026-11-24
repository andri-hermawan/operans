-- Ritase: productivity_factor dari numeric(12,4) menjadi text (isian bebas, bisa panjang).
ALTER TABLE core.ritases DROP CONSTRAINT ck_ritases_amounts;
ALTER TABLE core.ritases ALTER COLUMN productivity_factor TYPE text USING productivity_factor::text;
ALTER TABLE core.ritases ADD CONSTRAINT ck_ritases_amounts
    CHECK ((distance IS NULL OR distance >= 0) AND (ritase IS NULL OR ritase >= 0)
       AND (production IS NULL OR production >= 0) AND (production_adjusted IS NULL OR production_adjusted >= 0)
       AND (distance_volume IS NULL OR distance_volume >= 0));
