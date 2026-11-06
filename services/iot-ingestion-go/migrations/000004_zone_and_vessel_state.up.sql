-- C2 (ADR 0009): zona peta & status muatan.
--   zone_id                    = id zona geo.zones yang dipakai (lebih tepat dari geofence_fid/ORIG_FID yang tidak unik)
--   last_inside_fid / _vessel  = acuan status muatan: zona TERAKHIR yang ada di dalam peta (fid ≠ 0).
--                                Lama: query "log terakhir di dalam polygon" setiap paket; kini disimpan di live state
--                                agar tetap benar setelah layanan restart.
ALTER TABLE iot.telemetry_events      ADD COLUMN zone_id uuid;
ALTER TABLE iot.geofence_events       ADD COLUMN zone_id uuid;
ALTER TABLE iot.equipment_live_states
    ADD COLUMN zone_id                    uuid,
    ADD COLUMN last_inside_fid            integer,
    ADD COLUMN last_inside_vessel_status  varchar(20),
    ADD CONSTRAINT ck_live_states_last_inside_vessel CHECK (last_inside_vessel_status IN ('loaded', 'empty', 'unknown'));

COMMENT ON COLUMN iot.telemetry_events.zone_id IS 'Zona peta (geo.zones.id, core) tempat titik berada; NULL = di luar semua zona';
