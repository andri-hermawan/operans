-- 000014 (DOWN): menghapus hak akses halaman & laporan beserta isinya (kembali ke aturan bawaan).
DROP TRIGGER trg_project_members_drop_access_rules ON auth.project_members;
DROP FUNCTION auth.drop_user_access_rules();
DROP TABLE auth.access_rules;
