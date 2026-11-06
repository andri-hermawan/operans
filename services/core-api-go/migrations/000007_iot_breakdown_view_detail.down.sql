DROP VIEW core.v_iot_equipment_breakdowns;
CREATE VIEW core.v_iot_equipment_breakdowns AS
SELECT b.project_id, b.operational_date, s.shift_code, b.equipment_code, b.equipment_id, b.status,
       b.category_code, b.started_at, b.ended_at
  FROM core.equipment_breakdowns b JOIN core.shifts s ON s.id = b.shift_id
 WHERE b.operational_date >= current_date - 7;
GRANT SELECT ON core.v_iot_equipment_breakdowns TO iot_ingest, core_api;
