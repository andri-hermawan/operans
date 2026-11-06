-- 000002_search_index (UP)
-- Pencarian kata kunci (full-text) pada teks per halaman dokumen, memakai konfigurasi bahasa Indonesia
-- bawaan PostgreSQL (stemming: "keselamatan" ~ "selamat"). Rancangan: ADR 0013.
-- Kueri pencarian di app/search/repository.py HARUS memakai ekspresi yang sama, agar indeks ini terpakai.
CREATE INDEX idx_document_pages_fts ON ai.document_pages USING GIN (to_tsvector('indonesian', content));
