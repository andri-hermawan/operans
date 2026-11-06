-- Fase D/F2: halaman System Data → Device mengelola iot.devices lewat layanan IoT (pemilik tabel).
-- iot_ingest boleh menambah alat & mengubah kolom pengaturan; tidak boleh menghapus (riwayat telemetry menunjuk alat).
GRANT INSERT ON iot.devices TO iot_ingest;
GRANT UPDATE (name, equipment_id, project_id, model, fuel_source, sim_number, provider_name, api_key_hash, is_active, updated_at)
    ON iot.devices TO iot_ingest;
