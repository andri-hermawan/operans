-- 000001_init_iot (DOWN) — urutan kebalikan: tabel anak dulu, baru induk
DROP TABLE IF EXISTS iot.telemetry_events;
DROP TABLE IF EXISTS iot.devices;