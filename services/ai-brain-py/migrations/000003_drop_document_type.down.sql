-- 000003_drop_document_type (DOWN) — mengembalikan kolom jenis; baris yang ada dianggap 'sop'.
ALTER TABLE ai.documents ADD COLUMN document_type varchar(20) NOT NULL DEFAULT 'sop';
ALTER TABLE ai.documents ADD CONSTRAINT ck_documents_type CHECK (document_type IN ('sop', 'hse'));
ALTER TABLE ai.documents ALTER COLUMN document_type DROP DEFAULT;
