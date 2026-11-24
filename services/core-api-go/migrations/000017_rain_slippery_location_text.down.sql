-- Kembali ke varchar(100); gagal bila ada isi lebih panjang dari 100 karakter.
ALTER TABLE core.rain_slippery_logs ALTER COLUMN location TYPE varchar(100);
