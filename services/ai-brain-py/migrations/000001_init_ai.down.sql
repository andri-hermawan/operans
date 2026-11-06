-- 000001_init_ai (DOWN) — menghapus semua tabel AI Brain beserta isinya (dokumen, teks, riwayat chat, audit).
DROP TABLE ai.audit_logs;
DROP TABLE ai.chat_messages;
DROP TABLE ai.chat_sessions;
DROP TABLE ai.document_chunks;
DROP TABLE ai.document_pages;
DROP TABLE ai.document_roles;
DROP TABLE ai.documents;
DROP TABLE ai.departments;
DROP FUNCTION ai.touch_updated_at();
