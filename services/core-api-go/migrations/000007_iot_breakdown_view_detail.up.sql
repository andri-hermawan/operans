-- C1 (ADR 0009): IoT menampilkan keterangan breakdown di live state dan memilih catatan terbaru.
-- Kolom baru ditambahkan di AKHIR view (syarat CREATE OR REPLACE VIEW).
CREATE OR REPLACE VIEW core.v_iot_equipment_breakdowns AS
SELECT b.project_id, b.operational_date, s.shift_code, b.equipment_code, b.equipment_id, b.status,
       b.category_code, b.started_at, b.ended_at, b.description, b.updated_at
  FROM core.equipment_breakdowns b JOIN core.shifts s ON s.id = b.shift_id
 WHERE b.operational_date >= current_date - 7;
