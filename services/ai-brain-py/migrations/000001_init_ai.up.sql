-- 000001_init_ai (UP)
-- Prasyarat (admin): role ai_migrator/ai_app & schema ai — db/admin/001_roles_schema_ai.sql
-- Standar: ADR 0004 (penamaan Inggris snake_case, boolean is_…, waktu timestamptz). Rancangan: ADR 0013.
-- Kolom vektor (embedding) SENGAJA belum ada: ukurannya ikut model embedding, yang baru dipilih nanti
-- (migration terpisah). pgvector hanya untuk teks dokumen — telemetry tidak masuk sini.

-- =====================================================================
-- 0. updated_at otomatis
-- =====================================================================
CREATE FUNCTION ai.touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

-- =====================================================================
-- 1. ai.departments — pengelompokan dokumen per department (per project)
-- =====================================================================
CREATE TABLE ai.departments (
    id               uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id       uuid          NOT NULL,
    department_code  varchar(30)   NOT NULL,
    department_name  varchar(100)  NOT NULL,
    is_active        boolean       NOT NULL DEFAULT true,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    updated_at       timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_departments          PRIMARY KEY (id),
    CONSTRAINT fk_departments_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT uq_departments_code     UNIQUE (project_id, department_code)
);
CREATE TRIGGER trg_departments_touch BEFORE UPDATE ON ai.departments
    FOR EACH ROW EXECUTE FUNCTION ai.touch_updated_at();

-- =====================================================================
-- 2. ai.documents — satu baris per file SOP / HSE (PDF)
-- =====================================================================
CREATE TABLE ai.documents (
    id                 uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id         uuid          NOT NULL,
    department_id      uuid          NOT NULL,
    document_type      varchar(20)   NOT NULL,            -- sop | hse
    title              varchar(200)  NOT NULL,
    document_number    varchar(100),                      -- nomor dokumen resmi, bila ada
    revision           varchar(30),
    file_name          varchar(255)  NOT NULL,            -- nama asli saat diunggah
    file_sha256        char(64)      NOT NULL,            -- sidik jari isi file → unggah ulang file sama = tidak ganda
    file_size_bytes    bigint        NOT NULL,
    storage_path       varchar(500)  NOT NULL,            -- lokasi file di penyimpanan layanan
    page_count         integer,
    processing_status  varchar(20)   NOT NULL DEFAULT 'uploaded',  -- uploaded → processing → ready | failed
    processing_error   text,
    is_active          boolean       NOT NULL DEFAULT true,         -- false = ditarik (tidak dihapus, riwayat terjaga)
    uploaded_by        uuid,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    updated_at         timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_documents             PRIMARY KEY (id),
    CONSTRAINT fk_documents_project     FOREIGN KEY (project_id)    REFERENCES core.projects (id),
    CONSTRAINT fk_documents_department  FOREIGN KEY (department_id) REFERENCES ai.departments (id),
    CONSTRAINT fk_documents_uploader    FOREIGN KEY (uploaded_by)   REFERENCES auth.users (id),
    CONSTRAINT uq_documents_file        UNIQUE (project_id, file_sha256),
    CONSTRAINT ck_documents_type        CHECK (document_type IN ('sop', 'hse')),
    CONSTRAINT ck_documents_status      CHECK (processing_status IN ('uploaded', 'processing', 'ready', 'failed')),
    CONSTRAINT ck_documents_sha256      CHECK (file_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT ck_documents_size        CHECK (file_size_bytes > 0),
    CONSTRAINT ck_documents_pages       CHECK (page_count IS NULL OR page_count > 0)
);
CREATE INDEX idx_documents_project_dept ON ai.documents (project_id, department_id) WHERE is_active;
CREATE TRIGGER trg_documents_touch BEFORE UPDATE ON ai.documents
    FOR EACH ROW EXECUTE FUNCTION ai.touch_updated_at();

-- =====================================================================
-- 3. ai.document_roles — role yang boleh MEMBACA sebuah dokumen.
--    Admin project dan superadmin selalu boleh. Dokumen tanpa baris di sini hanya terlihat oleh admin.
--    Filter ini dijalankan di SQL (retrieval), sebelum teks sampai ke LLM.
-- =====================================================================
CREATE TABLE ai.document_roles (
    document_id  uuid         NOT NULL,
    role         varchar(20)  NOT NULL,

    CONSTRAINT pk_document_roles      PRIMARY KEY (document_id, role),
    CONSTRAINT fk_document_roles_doc  FOREIGN KEY (document_id) REFERENCES ai.documents (id) ON DELETE CASCADE,
    CONSTRAINT ck_document_roles_role CHECK (role IN ('dispatcher', 'viewer'))
);

-- =====================================================================
-- 4. ai.document_pages — teks hasil ekstraksi per halaman (dasar tampilan "baca per halaman" & sitasi)
-- =====================================================================
CREATE TABLE ai.document_pages (
    id           bigint   GENERATED ALWAYS AS IDENTITY,
    document_id  uuid     NOT NULL,
    page_number  integer  NOT NULL,
    content      text     NOT NULL,

    CONSTRAINT pk_document_pages      PRIMARY KEY (id),
    CONSTRAINT fk_document_pages_doc  FOREIGN KEY (document_id) REFERENCES ai.documents (id) ON DELETE CASCADE,
    CONSTRAINT uq_document_pages      UNIQUE (document_id, page_number),
    CONSTRAINT ck_document_pages_no   CHECK (page_number > 0)
);

-- =====================================================================
-- 5. ai.document_chunks — potongan teks untuk pencarian (kelak: + kolom embedding vector(N))
-- =====================================================================
CREATE TABLE ai.document_chunks (
    id               bigint        GENERATED ALWAYS AS IDENTITY,
    document_id      uuid          NOT NULL,
    chunk_index      integer       NOT NULL,
    page_from        integer       NOT NULL,
    page_to          integer       NOT NULL,
    content          text          NOT NULL,
    token_count      integer,
    embedding_model  varchar(100),                          -- NULL = belum di-embed; terisi bersama kolom vektor nanti
    created_at       timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_document_chunks      PRIMARY KEY (id),
    CONSTRAINT fk_document_chunks_doc  FOREIGN KEY (document_id) REFERENCES ai.documents (id) ON DELETE CASCADE,
    CONSTRAINT uq_document_chunks      UNIQUE (document_id, chunk_index),
    CONSTRAINT ck_document_chunks_page CHECK (page_from > 0 AND page_to >= page_from)
);

-- =====================================================================
-- 6. ai.chat_sessions & ai.chat_messages — riwayat tanya-jawab per user
-- =====================================================================
CREATE TABLE ai.chat_sessions (
    id          uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id  uuid          NOT NULL,
    user_id     uuid          NOT NULL,
    title       varchar(200),
    created_at  timestamptz   NOT NULL DEFAULT now(),
    updated_at  timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_chat_sessions          PRIMARY KEY (id),
    CONSTRAINT fk_chat_sessions_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT fk_chat_sessions_user     FOREIGN KEY (user_id)    REFERENCES auth.users (id)
);
CREATE INDEX idx_chat_sessions_user ON ai.chat_sessions (project_id, user_id, updated_at DESC);
CREATE TRIGGER trg_chat_sessions_touch BEFORE UPDATE ON ai.chat_sessions
    FOR EACH ROW EXECUTE FUNCTION ai.touch_updated_at();

CREATE TABLE ai.chat_messages (
    id          bigint        GENERATED ALWAYS AS IDENTITY,
    session_id  uuid          NOT NULL,
    role        varchar(10)   NOT NULL,                    -- user | assistant
    content     text          NOT NULL,
    sources     jsonb         NOT NULL DEFAULT '[]',       -- dokumen & halaman rujukan jawaban
    warnings    jsonb         NOT NULL DEFAULT '[]',
    request_id  varchar(64),
    created_at  timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_chat_messages          PRIMARY KEY (id),
    CONSTRAINT fk_chat_messages_session  FOREIGN KEY (session_id) REFERENCES ai.chat_sessions (id) ON DELETE CASCADE,
    CONSTRAINT ck_chat_messages_role     CHECK (role IN ('user', 'assistant'))
);
CREATE INDEX idx_chat_messages_session ON ai.chat_messages (session_id, id);

-- =====================================================================
-- 7. ai.audit_logs — siapa membuka/mencari apa. Hanya ditambah (tidak ada UPDATE/DELETE).
--    Tanpa foreign key: catatan audit tetap ada walau dokumen/user kelak dihapus.
-- =====================================================================
CREATE TABLE ai.audit_logs (
    id           bigint        GENERATED ALWAYS AS IDENTITY,
    project_id   uuid,
    user_id      uuid,
    action       varchar(50)   NOT NULL,                   -- mis. document_view, document_upload, search, chat
    document_id  uuid,
    detail       jsonb         NOT NULL DEFAULT '{}',
    request_id   varchar(64),
    created_at   timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT pk_audit_logs PRIMARY KEY (id)
);
CREATE INDEX idx_audit_logs_project ON ai.audit_logs (project_id, created_at DESC);
CREATE INDEX idx_audit_logs_user    ON ai.audit_logs (user_id, created_at DESC);

-- =====================================================================
-- 8. Hak akun layanan (minimum yang dibutuhkan)
-- =====================================================================
GRANT SELECT, INSERT, UPDATE         ON ai.departments, ai.documents          TO ai_app;  -- dokumen ditarik lewat is_active, bukan dihapus
GRANT SELECT, INSERT, DELETE         ON ai.document_roles, ai.document_pages  TO ai_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ai.document_chunks, ai.chat_sessions  TO ai_app;  -- chunk: diganti saat ingest ulang, diisi embedding
GRANT SELECT, INSERT                 ON ai.chat_messages, ai.audit_logs       TO ai_app;  -- audit: hanya ditambah
GRANT USAGE ON ALL SEQUENCES IN SCHEMA ai TO ai_app;
