-- AI Brain — persiapan database oleh ADMIN (dijalankan SEKALI sebagai postgres / DBA).
-- Pola sama dengan core-api (db/admin/001_roles_schemas.sql) dan layanan IoT (ADR 0001, 0013).
--
--   psql -h 192.168.2.47 -p 5432 -U postgres -d simha -f db/admin/001_roles_schema_ai.sql
--
-- Password TIDAK ditulis di file ini. Setelah file selesai dijalankan, pasang password di psql:
--   \password ai_migrator
--   \password ai_app

-- 1) Dua akun: pemilik schema (hanya untuk migration) dan akun layanan (hak minimum)
CREATE ROLE ai_migrator LOGIN;
CREATE ROLE ai_app      LOGIN CONNECTION LIMIT 20;

-- 2) Boleh masuk ke database simha
GRANT CONNECT ON DATABASE simha TO ai_migrator, ai_app;

-- 3) Schema milik AI Brain
CREATE SCHEMA ai AUTHORIZATION ai_migrator;
GRANT USAGE ON SCHEMA ai TO ai_app;

-- 4) Tabel AI merujuk project & user milik core-api (foreign key). Hanya hak "REFERENCES": tidak bisa membaca isinya.
GRANT USAGE ON SCHEMA core, auth TO ai_migrator;
GRANT REFERENCES ON core.projects, auth.users TO ai_migrator;
