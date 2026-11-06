-- Project khusus UJI OTOMATIS (tools\uji-*.ps1) agar uji tidak menyentuh data asli CHR/CMN/EMP
-- (sejak B8 CHR berisi peta, aturan alert & data harian asli). Akun uji admin & dispatcher menjadi
-- anggotanya; login tanpa project_id tetap ke CHR (urut kode project).
-- Hapus project ini (beserta datanya) sebelum produksi. Aman dijalankan berulang.
--
--   psql "<MIGRATE_DATABASE_URL tanpa search_path>" -f db/seed/002_uji.sql

INSERT INTO core.projects (id, company_id, project_code, project_name, timezone)
VALUES ('f0000000-0000-4000-8000-000000000001', '5006fabf-3b40-4fa2-9260-c43b9f8452e0',
        'UJI', 'Project Uji Otomatis', 'Asia/Jakarta')
ON CONFLICT DO NOTHING;

INSERT INTO core.shifts (project_id, shift_code, shift_name, start_time, end_time, sequence)
VALUES ('f0000000-0000-4000-8000-000000000001', 'SHIFT_1', 'Shift 1', '07:00', '19:00', 1),
       ('f0000000-0000-4000-8000-000000000001', 'SHIFT_2', 'Shift 2', '19:00', '07:00', 2)
ON CONFLICT DO NOTHING;

INSERT INTO auth.project_members (user_id, project_id, role)
SELECT u.id, 'f0000000-0000-4000-8000-000000000001', r.role
  FROM auth.users u
  JOIN (VALUES ('uji.admin@simha.local', 'admin'), ('uji.dispatcher@simha.local', 'dispatcher')) r(email, role)
    ON r.email = u.email
ON CONFLICT DO NOTHING;
