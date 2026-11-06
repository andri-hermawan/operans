-- 000003_drop_document_type (UP)
-- Pustaka dokumen AI Brain khusus SOP. Dokumen HSE adalah modul lain (belum dirancang), jadi pembedaan
-- jenis (sop | hse) dilepas. Bila kelak ada jenis lain, kolom ditambah kembali lewat migration baru.
-- Tabel masih kosong saat migration ini dibuat; bila sudah berisi, semua baris otomatis dianggap SOP.
ALTER TABLE ai.documents DROP CONSTRAINT ck_documents_type;
ALTER TABLE ai.documents DROP COLUMN document_type;
