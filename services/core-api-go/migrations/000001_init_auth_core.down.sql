-- 000001_init_auth_core (DOWN) — urutan kebalikan: tabel anak dulu, baru induk
DROP TABLE IF EXISTS auth.refresh_tokens;
DROP TABLE IF EXISTS auth.project_members;
DROP TABLE IF EXISTS auth.users;
DROP TABLE IF EXISTS core.shifts;
DROP TABLE IF EXISTS core.projects;
DROP TABLE IF EXISTS core.companies;
DROP FUNCTION IF EXISTS core.touch_updated_at();
