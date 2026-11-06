-- core-api — persiapan database oleh ADMIN (dijalankan SEKALI sebagai postgres / DBA).
-- Pola sama dengan Step 4 layanan IoT (ADR 0001, 0006).
--
--   psql -h 192.168.2.47 -p 5432 -U postgres -d simha -f db/admin/001_roles_schemas.sql
--
-- Password TIDAK ditulis di file ini. Setelah file selesai dijalankan, pasang password di psql:
--   \password core_migrator
--   \password core_api

-- 1) Dua akun: pemilik schema (hanya untuk migration) dan akun layanan (hak minimum)
CREATE ROLE core_migrator LOGIN;
CREATE ROLE core_api      LOGIN CONNECTION LIMIT 20;

-- 2) Boleh masuk ke database simha (pintu sudah ditutup untuk PUBLIC di Step 4)
GRANT CONNECT ON DATABASE simha TO core_migrator, core_api;

-- 3) Schema milik core-api
CREATE SCHEMA auth AUTHORIZATION core_migrator;  -- user, keanggotaan project, refresh token
CREATE SCHEMA core AUTHORIZATION core_migrator;  -- data induk & input harian
CREATE SCHEMA geo  AUTHORIZATION core_migrator;  -- peta / zona (Fase B6)

GRANT USAGE ON SCHEMA auth, core, geo TO core_api;

-- 4) Layanan IoT kelak MEMBACA data induk lewat view di schema core/geo (Fase B8)
GRANT USAGE ON SCHEMA core, geo TO iot_ingest;
