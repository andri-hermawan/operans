-- 000001_init_auth_core (UP)
-- Prasyarat (admin): role core_migrator/core_api, schema auth/core/geo — db/admin/001_roles_schemas.sql
-- Standar: ADR 0004 (penamaan, kode baku, waktu). Rancangan: ADR 0002 & 0006.
-- Pengganti tabel fms_lite: companies, projects, shifts, users (+ role), auth_sessions.

-- =====================================================================
-- 0. updated_at otomatis — tidak ada kode yang bisa "lupa" mengisinya
-- =====================================================================
CREATE FUNCTION core.touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

-- =====================================================================
-- 1. core.companies (lama: companies)
-- =====================================================================
CREATE TABLE core.companies (
    id             uuid          NOT NULL DEFAULT gen_random_uuid(),
    company_code   varchar(50)   NOT NULL,
    company_name   varchar(150)  NOT NULL,
    is_active      boolean       NOT NULL DEFAULT true,   -- lama: status 'active'
    created_at     timestamptz   NOT NULL DEFAULT now(),
    created_by     uuid,
    updated_at     timestamptz   NOT NULL DEFAULT now(),
    updated_by     uuid,

    CONSTRAINT pk_companies       PRIMARY KEY (id),
    CONSTRAINT uq_companies_code  UNIQUE (company_code)
);

-- =====================================================================
-- 2. core.projects (lama: projects). Peta (geojson_origin, geom_origin) pindah ke schema geo (B6).
-- =====================================================================
CREATE TABLE core.projects (
    id             uuid          NOT NULL DEFAULT gen_random_uuid(),
    company_id     uuid          NOT NULL,
    project_code   varchar(50)   NOT NULL,
    project_name   varchar(150)  NOT NULL,
    timezone       varchar(50)   NOT NULL DEFAULT 'Asia/Jakarta',  -- dasar jam operasional & shift (ADR 0004)
    image          text,                                           -- logo/gambar project (lama: image)
    is_active      boolean       NOT NULL DEFAULT true,
    created_at     timestamptz   NOT NULL DEFAULT now(),
    created_by     uuid,
    updated_at     timestamptz   NOT NULL DEFAULT now(),
    updated_by     uuid,

    CONSTRAINT pk_projects          PRIMARY KEY (id),
    CONSTRAINT uq_projects_code     UNIQUE (project_code),
    CONSTRAINT fk_projects_company  FOREIGN KEY (company_id) REFERENCES core.companies (id)
);

-- =====================================================================
-- 3. core.shifts (lama: shifts) — jam lokal project, boleh melewati tengah malam
-- =====================================================================
CREATE TABLE core.shifts (
    id             uuid          NOT NULL DEFAULT gen_random_uuid(),
    project_id     uuid          NOT NULL,
    shift_code     varchar(20)   NOT NULL,       -- SHIFT_1, SHIFT_2
    shift_name     varchar(100)  NOT NULL,       -- "Shift 1"
    start_time     time          NOT NULL,       -- 07:00
    end_time       time          NOT NULL,       -- 19:00 (lebih kecil dari start_time = melewati tengah malam)
    sequence       smallint      NOT NULL,       -- urutan dalam satu hari operasional
    is_active      boolean       NOT NULL DEFAULT true,
    created_at     timestamptz   NOT NULL DEFAULT now(),
    created_by     uuid,
    updated_at     timestamptz   NOT NULL DEFAULT now(),
    updated_by     uuid,

    CONSTRAINT pk_shifts               PRIMARY KEY (id),
    CONSTRAINT uq_shifts_project_code  UNIQUE (project_id, shift_code),
    CONSTRAINT uq_shifts_project_seq   UNIQUE (project_id, sequence),
    CONSTRAINT fk_shifts_project       FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT ck_shifts_duration      CHECK (start_time <> end_time)
);

-- =====================================================================
-- 4. auth.users (lama: users) — role per project dipisah ke auth.project_members
-- =====================================================================
CREATE TABLE auth.users (
    id                   uuid          NOT NULL DEFAULT gen_random_uuid(),
    email                varchar(150)  NOT NULL,
    name                 varchar(150)  NOT NULL,
    phone_number         varchar(20),                    -- dipakai notifikasi WhatsApp (Fase C)
    password_hash        text          NOT NULL,         -- argon2id; hash bcrypt lama diterima & diganti saat login
    is_superadmin        boolean       NOT NULL DEFAULT false,
    is_active            boolean       NOT NULL DEFAULT true,
    failed_login_count   integer       NOT NULL DEFAULT 0,  -- batas percobaan login (B2)
    locked_until         timestamptz,
    last_login_at        timestamptz,
    password_changed_at  timestamptz,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    created_by           uuid,
    updated_at           timestamptz   NOT NULL DEFAULT now(),
    updated_by           uuid,

    CONSTRAINT pk_users        PRIMARY KEY (id),
    CONSTRAINT ck_users_email  CHECK (email = lower(btrim(email)) AND email LIKE '%_@_%')
);
CREATE UNIQUE INDEX uq_users_email ON auth.users (email);

-- =====================================================================
-- 5. auth.project_members — role SETIAP user di SETIAP project (ADR 0002)
-- =====================================================================
CREATE TABLE auth.project_members (
    user_id        uuid          NOT NULL,
    project_id     uuid          NOT NULL,
    role           varchar(20)   NOT NULL,
    created_at     timestamptz   NOT NULL DEFAULT now(),
    created_by     uuid,
    updated_at     timestamptz   NOT NULL DEFAULT now(),
    updated_by     uuid,

    CONSTRAINT pk_project_members          PRIMARY KEY (user_id, project_id),
    CONSTRAINT fk_project_members_user     FOREIGN KEY (user_id)    REFERENCES auth.users (id),
    CONSTRAINT fk_project_members_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT ck_project_members_role     CHECK (role IN ('admin', 'dispatcher', 'viewer'))
);
CREATE INDEX idx_project_members_project ON auth.project_members (project_id);

-- =====================================================================
-- 6. auth.refresh_tokens (lama: auth_sessions — yang menyimpan token ASLI; di sini hanya hash)
--    Rotation: setiap refresh menghasilkan token baru dalam "keluarga" (family_id) yang sama.
--    Token yang sudah dipakai lalu dipakai lagi = dicuri → seluruh keluarga dicabut.
-- =====================================================================
CREATE TABLE auth.refresh_tokens (
    id                   uuid          NOT NULL DEFAULT gen_random_uuid(),
    family_id            uuid          NOT NULL,       -- satu sesi login = satu keluarga
    user_id              uuid          NOT NULL,
    project_id           uuid,                         -- project aktif sesi ini (NULL: superadmin tanpa project)
    token_hash           varchar(64)   NOT NULL,       -- SHA-256 hex; token asli tidak pernah disimpan
    issued_at            timestamptz   NOT NULL DEFAULT now(),
    expires_at           timestamptz   NOT NULL,       -- 7 hari tidak dipakai → habis
    absolute_expires_at  timestamptz   NOT NULL,       -- maks. 30 hari sejak login
    used_at              timestamptz,                  -- sudah ditukar dengan token baru
    replaced_by          uuid,
    revoked_at           timestamptz,
    revoked_reason       varchar(50),                  -- logout | reuse_detected | admin | password_changed
    user_agent           varchar(255),
    ip_address           varchar(45),

    CONSTRAINT pk_refresh_tokens          PRIMARY KEY (id),
    CONSTRAINT uq_refresh_tokens_hash     UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user     FOREIGN KEY (user_id)    REFERENCES auth.users (id),
    CONSTRAINT fk_refresh_tokens_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT ck_refresh_tokens_expiry   CHECK (expires_at <= absolute_expires_at)
);
CREATE INDEX idx_refresh_tokens_user   ON auth.refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_family ON auth.refresh_tokens (family_id);

-- =====================================================================
-- Trigger updated_at
-- =====================================================================
CREATE TRIGGER trg_companies_updated_at       BEFORE UPDATE ON core.companies       FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_projects_updated_at        BEFORE UPDATE ON core.projects        FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_shifts_updated_at          BEFORE UPDATE ON core.shifts          FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_users_updated_at           BEFORE UPDATE ON auth.users           FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();
CREATE TRIGGER trg_project_members_updated_at BEFORE UPDATE ON auth.project_members FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

-- =====================================================================
-- Hak akses minimum untuk layanan core_api
-- Data induk tidak dihapus (cukup is_active = false), kecuali keanggotaan & token.
-- =====================================================================
GRANT SELECT, INSERT, UPDATE         ON core.companies       TO core_api;
GRANT SELECT, INSERT, UPDATE         ON core.projects        TO core_api;
GRANT SELECT, INSERT, UPDATE         ON core.shifts          TO core_api;
GRANT SELECT, INSERT, UPDATE         ON auth.users           TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.project_members TO core_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON auth.refresh_tokens  TO core_api;
GRANT EXECUTE ON FUNCTION core.touch_updated_at() TO core_api;
