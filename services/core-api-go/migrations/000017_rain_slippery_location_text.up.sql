-- Rain & Slippery: location bisa memuat beberapa segment (dipisah koma), jadi tipe text.
ALTER TABLE core.rain_slippery_logs ALTER COLUMN location TYPE text;
