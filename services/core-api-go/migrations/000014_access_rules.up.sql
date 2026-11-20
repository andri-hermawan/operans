-- Hak akses halaman & laporan yang diatur superadmin (ADR 0018). Per project, lalu per role ATAU per user.
-- Urutan hak akhir: superadmin (selalu penuh) → aturan user → aturan role → aturan bawaan di kode (frontend/backend).
-- Baris hanya ada untuk yang DIUBAH dari bawaan; "allowed" = true (izinkan) / false (blokir).

CREATE TABLE auth.access_rules (
    id            bigint        GENERATED ALWAYS AS IDENTITY,
    project_id    uuid          NOT NULL,
    subject_type  varchar(10)   NOT NULL,              -- 'role' | 'user'
    role          varchar(20),                         -- terisi bila subject_type = 'role'
    user_id       uuid,                                -- terisi bila subject_type = 'user'
    page_key      varchar(80)   NOT NULL,              -- kunci halaman/laporan, mis. 'report-ritase-daily'
    allowed       boolean       NOT NULL,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    created_by    uuid,
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    updated_by    uuid,

    CONSTRAINT pk_access_rules          PRIMARY KEY (id),
    CONSTRAINT fk_access_rules_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT fk_access_rules_user     FOREIGN KEY (user_id)    REFERENCES auth.users (id),
    CONSTRAINT ck_access_rules_subject  CHECK (
        (subject_type = 'role' AND user_id IS NULL AND role IN ('admin', 'dispatcher', 'viewer'))
     OR (subject_type = 'user' AND user_id IS NOT NULL AND role IS NULL)
    ),
    CONSTRAINT ck_access_rules_key      CHECK (page_key <> '' AND page_key = lower(btrim(page_key)))
);

-- satu aturan per (project, subjek, halaman)
CREATE UNIQUE INDEX uq_access_rules_role ON auth.access_rules (project_id, role, page_key)    WHERE subject_type = 'role';
CREATE UNIQUE INDEX uq_access_rules_user ON auth.access_rules (project_id, user_id, page_key) WHERE subject_type = 'user';
CREATE INDEX idx_access_rules_user ON auth.access_rules (user_id) WHERE user_id IS NOT NULL;

CREATE TRIGGER trg_access_rules_updated_at BEFORE UPDATE ON auth.access_rules FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

-- User dikeluarkan dari project → aturan khusus user di project itu ikut dihapus.
CREATE FUNCTION auth.drop_user_access_rules() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM auth.access_rules WHERE project_id = OLD.project_id AND user_id = OLD.user_id;
    RETURN OLD;
END;
$$;
CREATE TRIGGER trg_project_members_drop_access_rules AFTER DELETE ON auth.project_members FOR EACH ROW EXECUTE FUNCTION auth.drop_user_access_rules();

GRANT SELECT, INSERT, UPDATE, DELETE ON auth.access_rules TO core_api;
