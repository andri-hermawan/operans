-- Status muatan (vessel_status) hanya berlaku untuk Dump Truck (keputusan 2026-11-17).
-- Unit lain (Water Truck, Fuel Truck, …): di dalam zona → kosong (NULL); di luar zona (off-track) → 'unknown'.
-- Unit tanpa tipe diperlakukan seperti Dump Truck (tidak diubah). Kejadian masuk/keluar zona tetap dicatat.
-- Daftar unit non-Dump Truck ditulis langsung (16 unit per 2026-11-17: Water Truck, Fuel Truck, Excavator) karena role migrasi iot
-- tidak boleh membaca schema core. Unit yang ditambah/diubah tipenya setelah ini dihitung kode baru, bukan migrasi ini.
-- Mengubah DATA LAMA; kode baru (iot-ingestion-go) sudah memakai aturan yang sama untuk data baru.

UPDATE iot.telemetry_events t
   SET vessel_status = CASE WHEN t.is_inside_geofence IS FALSE THEN 'unknown' END
 WHERE t.equipment_id IN (SELECT unnest(ARRAY['48258b4f-2bef-43c6-a78b-7ca2eaf6ee07'::uuid, '1c879854-a0e2-494a-a552-5390290b042f'::uuid, '394bf3ba-4aeb-49d3-b171-883353534a21'::uuid, 'aec4bd8b-b871-44d4-bae8-85869879df37'::uuid, '38e07f1e-c919-40e3-b142-df4575cf8202'::uuid, 'a0a03e19-3663-415d-884f-dbd0fdc4ca94'::uuid, '0250c1ea-51f4-4d20-ab51-480e8d0c32ff'::uuid, '4dbcc3c6-df4d-429e-a236-5df69e16789e'::uuid, 'e8e07be6-4289-4a9f-a860-2875e632aac4'::uuid, '618e0331-0a5c-40ab-b0b3-41cd4253fb58'::uuid, '2034444d-f536-4300-a1d1-eea0fdeb4916'::uuid, 'd4e3b45e-44ff-4381-ac9e-410f456d8045'::uuid, '91ef3c87-f33f-443f-b3c9-64a56711255e'::uuid, '7d4b1149-b238-46a0-a43d-60fd783bef59'::uuid, '6212a126-c266-44c8-bf3e-4a7f1479b526'::uuid, '08f2e3e0-aae2-490d-8c2e-197bc35620ae'::uuid]))
   AND t.vessel_status IS DISTINCT FROM (CASE WHEN t.is_inside_geofence IS FALSE THEN 'unknown' END);

UPDATE iot.equipment_live_states s
   SET vessel_status = CASE WHEN s.is_inside_geofence IS FALSE THEN 'unknown' END,
       last_inside_fid = NULL, last_inside_vessel_status = NULL
 WHERE s.equipment_id IN (SELECT unnest(ARRAY['48258b4f-2bef-43c6-a78b-7ca2eaf6ee07'::uuid, '1c879854-a0e2-494a-a552-5390290b042f'::uuid, '394bf3ba-4aeb-49d3-b171-883353534a21'::uuid, 'aec4bd8b-b871-44d4-bae8-85869879df37'::uuid, '38e07f1e-c919-40e3-b142-df4575cf8202'::uuid, 'a0a03e19-3663-415d-884f-dbd0fdc4ca94'::uuid, '0250c1ea-51f4-4d20-ab51-480e8d0c32ff'::uuid, '4dbcc3c6-df4d-429e-a236-5df69e16789e'::uuid, 'e8e07be6-4289-4a9f-a860-2875e632aac4'::uuid, '618e0331-0a5c-40ab-b0b3-41cd4253fb58'::uuid, '2034444d-f536-4300-a1d1-eea0fdeb4916'::uuid, 'd4e3b45e-44ff-4381-ac9e-410f456d8045'::uuid, '91ef3c87-f33f-443f-b3c9-64a56711255e'::uuid, '7d4b1149-b238-46a0-a43d-60fd783bef59'::uuid, '6212a126-c266-44c8-bf3e-4a7f1479b526'::uuid, '08f2e3e0-aae2-490d-8c2e-197bc35620ae'::uuid]));

UPDATE iot.fuel_events f
   SET vessel_status = CASE WHEN f.is_inside_geofence IS FALSE THEN 'unknown' END
 WHERE f.equipment_id IN (SELECT unnest(ARRAY['48258b4f-2bef-43c6-a78b-7ca2eaf6ee07'::uuid, '1c879854-a0e2-494a-a552-5390290b042f'::uuid, '394bf3ba-4aeb-49d3-b171-883353534a21'::uuid, 'aec4bd8b-b871-44d4-bae8-85869879df37'::uuid, '38e07f1e-c919-40e3-b142-df4575cf8202'::uuid, 'a0a03e19-3663-415d-884f-dbd0fdc4ca94'::uuid, '0250c1ea-51f4-4d20-ab51-480e8d0c32ff'::uuid, '4dbcc3c6-df4d-429e-a236-5df69e16789e'::uuid, 'e8e07be6-4289-4a9f-a860-2875e632aac4'::uuid, '618e0331-0a5c-40ab-b0b3-41cd4253fb58'::uuid, '2034444d-f536-4300-a1d1-eea0fdeb4916'::uuid, 'd4e3b45e-44ff-4381-ac9e-410f456d8045'::uuid, '91ef3c87-f33f-443f-b3c9-64a56711255e'::uuid, '7d4b1149-b238-46a0-a43d-60fd783bef59'::uuid, '6212a126-c266-44c8-bf3e-4a7f1479b526'::uuid, '08f2e3e0-aae2-490d-8c2e-197bc35620ae'::uuid]));

UPDATE iot.alerts a
   SET vessel_status = CASE WHEN a.is_inside_geofence IS FALSE THEN 'unknown' END
 WHERE a.equipment_id IN (SELECT unnest(ARRAY['48258b4f-2bef-43c6-a78b-7ca2eaf6ee07'::uuid, '1c879854-a0e2-494a-a552-5390290b042f'::uuid, '394bf3ba-4aeb-49d3-b171-883353534a21'::uuid, 'aec4bd8b-b871-44d4-bae8-85869879df37'::uuid, '38e07f1e-c919-40e3-b142-df4575cf8202'::uuid, 'a0a03e19-3663-415d-884f-dbd0fdc4ca94'::uuid, '0250c1ea-51f4-4d20-ab51-480e8d0c32ff'::uuid, '4dbcc3c6-df4d-429e-a236-5df69e16789e'::uuid, 'e8e07be6-4289-4a9f-a860-2875e632aac4'::uuid, '618e0331-0a5c-40ab-b0b3-41cd4253fb58'::uuid, '2034444d-f536-4300-a1d1-eea0fdeb4916'::uuid, 'd4e3b45e-44ff-4381-ac9e-410f456d8045'::uuid, '91ef3c87-f33f-443f-b3c9-64a56711255e'::uuid, '7d4b1149-b238-46a0-a43d-60fd783bef59'::uuid, '6212a126-c266-44c8-bf3e-4a7f1479b526'::uuid, '08f2e3e0-aae2-490d-8c2e-197bc35620ae'::uuid]));

-- kejadian zona = selalu terjadi di dalam/pinggir zona → kosong
UPDATE iot.geofence_events g
   SET vessel_status = NULL
 WHERE g.equipment_id IN (SELECT unnest(ARRAY['48258b4f-2bef-43c6-a78b-7ca2eaf6ee07'::uuid, '1c879854-a0e2-494a-a552-5390290b042f'::uuid, '394bf3ba-4aeb-49d3-b171-883353534a21'::uuid, 'aec4bd8b-b871-44d4-bae8-85869879df37'::uuid, '38e07f1e-c919-40e3-b142-df4575cf8202'::uuid, 'a0a03e19-3663-415d-884f-dbd0fdc4ca94'::uuid, '0250c1ea-51f4-4d20-ab51-480e8d0c32ff'::uuid, '4dbcc3c6-df4d-429e-a236-5df69e16789e'::uuid, 'e8e07be6-4289-4a9f-a860-2875e632aac4'::uuid, '618e0331-0a5c-40ab-b0b3-41cd4253fb58'::uuid, '2034444d-f536-4300-a1d1-eea0fdeb4916'::uuid, 'd4e3b45e-44ff-4381-ac9e-410f456d8045'::uuid, '91ef3c87-f33f-443f-b3c9-64a56711255e'::uuid, '7d4b1149-b238-46a0-a43d-60fd783bef59'::uuid, '6212a126-c266-44c8-bf3e-4a7f1479b526'::uuid, '08f2e3e0-aae2-490d-8c2e-197bc35620ae'::uuid])) AND g.vessel_status IS NOT NULL;
