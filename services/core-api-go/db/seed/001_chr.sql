-- Data awal project CHR, memakai UUID yang SAMA dengan fms_lite (ADR 0002/0006).
-- Aman dijalankan berulang (ON CONFLICT DO NOTHING). Impor lengkap dari fms_lite menyusul di B8.
--
--   psql "<DATABASE_URL core_api>" -f db/seed/001_chr.sql

INSERT INTO core.companies (id, company_code, company_name)
VALUES ('5006fabf-3b40-4fa2-9260-c43b9f8452e0', 'RMK', 'PT. RMK')
ON CONFLICT DO NOTHING;

INSERT INTO core.projects (id, company_id, project_code, project_name, timezone)
VALUES ('9811d93d-0813-489a-805d-eea4b68b3242', '5006fabf-3b40-4fa2-9260-c43b9f8452e0',
        'CHR', 'Coal Hauling Road', 'Asia/Jakarta')
ON CONFLICT DO NOTHING;

INSERT INTO core.shifts (project_id, shift_code, shift_name, start_time, end_time, sequence)
VALUES ('9811d93d-0813-489a-805d-eea4b68b3242', 'SHIFT_1', 'Shift 1', '07:00', '19:00', 1),
       ('9811d93d-0813-489a-805d-eea4b68b3242', 'SHIFT_2', 'Shift 2', '19:00', '07:00', 2)
ON CONFLICT DO NOTHING;
