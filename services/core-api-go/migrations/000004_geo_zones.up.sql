-- B6 (ADR 0003): peta project sebagai zona spasial berversi.
-- Lama: projects.geojson_origin (satu GeoJSON besar ikut respons login).
-- Baru: arsip file asli per upload (geo.layer_imports) + satu baris per feature (geo.zones).
-- Upload baru TIDAK menghapus zona lama: valid_to zona lama diisi → riwayat peta terjaga.

-- =====================================================================
-- 1. geo.layer_imports — arsip setiap file GeoJSON yang diunggah
-- =====================================================================
CREATE TABLE geo.layer_imports (
    id             uuid           NOT NULL DEFAULT gen_random_uuid(),
    project_id     uuid           NOT NULL,
    file_name      varchar(255)   NOT NULL,
    raw_geojson    jsonb          NOT NULL,   -- file asli (bisa diunduh ulang)
    feature_count  integer        NOT NULL,
    note           varchar(255),
    imported_by    uuid,
    imported_at    timestamptz    NOT NULL DEFAULT now(),

    CONSTRAINT pk_layer_imports          PRIMARY KEY (id),
    CONSTRAINT fk_layer_imports_project  FOREIGN KEY (project_id)  REFERENCES core.projects (id),
    CONSTRAINT fk_layer_imports_user     FOREIGN KEY (imported_by) REFERENCES auth.users (id),
    CONSTRAINT ck_layer_imports_count    CHECK (feature_count >= 0)
);
CREATE INDEX idx_layer_imports_project ON geo.layer_imports (project_id, imported_at DESC);

-- =====================================================================
-- 2. geo.zones — satu baris per feature (area)
-- =====================================================================
CREATE TABLE geo.zones (
    id           uuid                          NOT NULL DEFAULT gen_random_uuid(),
    project_id   uuid                          NOT NULL,
    import_id    uuid                          NOT NULL,
    source_fid   integer,                                  -- lama: ORIG_FID (tidak unik)
    category     varchar(100)                  NOT NULL,   -- lama: Category (mis. Trase Utama)
    name         varchar(100)                  NOT NULL,   -- lama: Segment  (mis. Km 23+000 - 24+000)
    line_color   varchar(9),                               -- lama: HEX_Line
    fill_color   varchar(9),                               -- lama: HEX_Fill
    priority     smallint                      NOT NULL DEFAULT 0,  -- zona bertumpuk: lebih tinggi menang
    geom         geometry(MultiPolygon, 4326)  NOT NULL,
    area_m2      numeric(14,2)                 NOT NULL,   -- zona bertumpuk, priority sama: terkecil menang
    properties   jsonb                         NOT NULL DEFAULT '{}',  -- semua atribut asli (BUFF_DIST, CODE, …)
    valid_from   timestamptz                   NOT NULL,
    valid_to     timestamptz,                              -- NULL = peta yang berlaku sekarang
    created_at   timestamptz                   NOT NULL DEFAULT now(),
    created_by   uuid,

    CONSTRAINT pk_zones          PRIMARY KEY (id),
    CONSTRAINT fk_zones_project  FOREIGN KEY (project_id) REFERENCES core.projects (id),
    CONSTRAINT fk_zones_import   FOREIGN KEY (import_id)  REFERENCES geo.layer_imports (id),
    CONSTRAINT ck_zones_period   CHECK (valid_to IS NULL OR valid_to >= valid_from),
    CONSTRAINT ck_zones_colors   CHECK ((line_color IS NULL OR line_color ~ '^#[0-9A-F]{6}([0-9A-F]{2})?$')
                                    AND (fill_color IS NULL OR fill_color ~ '^#[0-9A-F]{6}([0-9A-F]{2})?$')),
    CONSTRAINT ck_zones_geom     CHECK (ST_IsValid(geom) AND NOT ST_IsEmpty(geom))
);
CREATE INDEX idx_zones_geom     ON geo.zones USING GIST (geom);
CREATE INDEX idx_zones_current  ON geo.zones (project_id, category) WHERE valid_to IS NULL;
CREATE INDEX idx_zones_period   ON geo.zones (project_id, valid_from, valid_to);
CREATE INDEX idx_zones_import   ON geo.zones (import_id);

-- =====================================================================
-- 3. geo.find_zone — zona tempat sebuah titik berada pada waktu tertentu.
--    Zona bertumpuk (mis. CSA di atas Trase Utama): priority tertinggi, lalu luas terkecil.
--    Dipakai core-api sekarang dan enrichment telemetry (Fase C).
-- =====================================================================
CREATE FUNCTION geo.find_zone(p_project uuid, p_lon double precision, p_lat double precision,
                              p_at timestamptz DEFAULT now())
RETURNS SETOF geo.zones
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT z.*
      FROM geo.zones z
     WHERE z.project_id = p_project
       AND z.valid_from <= p_at
       AND (z.valid_to IS NULL OR z.valid_to > p_at)
       AND ST_Intersects(z.geom, ST_SetSRID(ST_MakePoint(p_lon, p_lat), 4326))
     ORDER BY z.priority DESC, z.area_m2, z.name
     LIMIT 1
$$;

GRANT SELECT, INSERT ON geo.layer_imports TO core_api;
GRANT SELECT, INSERT, UPDATE ON geo.zones TO core_api;
GRANT EXECUTE ON FUNCTION geo.find_zone(uuid, double precision, double precision, timestamptz) TO core_api;
