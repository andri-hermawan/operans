-- Stock Emplacement: stok cargo harian per partner (lama: tabel StockEmplacement di SQL Server).
-- Satuan ton (angka tanpa akhiran _ton di nama kolom). Satu baris per partner per tanggal operasional;
-- Shift 1 / Shift 2 tetap kolom (bukan baris per shift). Kolom total disimpan apa adanya dari input.
-- Kolom Status lama (1 = baru, 2 = ubah) dan User/CreateDate/UpdateDate digantikan created_*/updated_*.
CREATE TABLE core.stock_emplacements (
    id                       bigint GENERATED ALWAYS AS IDENTITY,
    partner                  varchar(50) NOT NULL,
    operational_date         date NOT NULL,
    stock_cargo_initial      numeric(14,2),
    stock_cargo_scrapper     numeric(14,2),
    stock_cargo_crusher      numeric(14,2),
    info                     varchar(10),
    shift_1                  numeric(14,2),
    shift_2                  numeric(14,2),
    hauling_cargo            numeric(14,2),
    total_cargo_initial      numeric(14,2),
    tls_out_trainset_1       numeric(14,2),
    tls_out_trainset_2       numeric(14,2),
    tls_out_trainset_3       numeric(14,2),
    tls_out_trainset_4       numeric(14,2),
    tls_out_trainset_5       numeric(14,2),
    total_tls_out            numeric(14,2),
    remaining_cargo          numeric(14,2),
    created_at               timestamptz NOT NULL DEFAULT now(),
    created_by               uuid,
    updated_at               timestamptz NOT NULL DEFAULT now(),
    updated_by               uuid,
    CONSTRAINT pk_stock_emplacements PRIMARY KEY (id),
    CONSTRAINT uq_stock_emplacements_key UNIQUE (partner, operational_date)
);

CREATE TRIGGER trg_stock_emplacements_updated_at BEFORE UPDATE ON core.stock_emplacements
    FOR EACH ROW EXECUTE FUNCTION core.touch_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON core.stock_emplacements TO core_api;
