REVOKE UPDATE (name, equipment_id, project_id, model, fuel_source, sim_number, provider_name, api_key_hash, is_active, updated_at)
    ON iot.devices FROM iot_ingest;
REVOKE INSERT ON iot.devices FROM iot_ingest;
