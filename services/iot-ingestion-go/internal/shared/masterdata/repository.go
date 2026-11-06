package masterdata

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------
// Data induk dari core: HANYA lewat view core.v_iot_* (role iot_ingest tidak bisa membaca tabel core).
// ---------------------------------------------------------------------

// Repository = semua SQL pembaca data induk (memenuhi interface Store milik Cache).
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) LoadShifts(ctx context.Context) ([]ShiftRow, error) {
	return collectRows(ctx, r, `
		SELECT s.project_id, p.project_code, p.project_name, s.timezone, s.shift_code, s.shift_name,
		       to_char(s.start_time, 'HH24:MI'), to_char(s.end_time, 'HH24:MI'), s.sequence
		  FROM core.v_iot_shifts s JOIN core.v_iot_projects p ON p.project_id = s.project_id
		 WHERE p.is_active
		 ORDER BY s.project_id, s.sequence`,
		func(row pgx.Rows, x *ShiftRow) error {
			return row.Scan(&x.ProjectID, &x.ProjectCode, &x.ProjectName, &x.Timezone, &x.Code, &x.Name, &x.StartTime, &x.EndTime, &x.Sequence)
		})
}

func (r *Repository) LoadEquipment(ctx context.Context) ([]EquipmentRow, error) {
	return collectRows(ctx, r, `SELECT equipment_id, project_id, equipment_code FROM core.v_iot_equipment`,
		func(row pgx.Rows, x *EquipmentRow) error {
			return row.Scan(&x.ID, &x.ProjectID, &x.Code)
		})
}

func (r *Repository) LoadOperators(ctx context.Context) ([]OperatorRow, error) {
	return collectRows(ctx, r, `
		SELECT project_id, operational_date::text, shift_code, equipment_code, operator_name
		  FROM core.v_iot_operator_assignments`,
		func(row pgx.Rows, x *OperatorRow) error {
			return row.Scan(&x.ProjectID, &x.OperationalDate, &x.ShiftCode, &x.EquipmentCode, &x.OperatorName)
		})
}

func (r *Repository) LoadBreakdowns(ctx context.Context) ([]BreakdownRow, error) {
	return collectRows(ctx, r, `
		SELECT project_id, operational_date::text, shift_code, equipment_code, status, started_at, ended_at,
		       description, updated_at
		  FROM core.v_iot_equipment_breakdowns`,
		func(row pgx.Rows, x *BreakdownRow) error {
			return row.Scan(&x.ProjectID, &x.OperationalDate, &x.ShiftCode, &x.EquipmentCode, &x.Status,
				&x.StartedAt, &x.EndedAt, &x.Description, &x.UpdatedAt)
		})
}

func collectRows[T any](ctx context.Context, r *Repository, sql string, scan func(pgx.Rows, *T) error) ([]T, error) {
	rows, err := r.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		var x T
		if err := scan(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------
// Pendengar NOTIFY (koneksi khusus, di luar pool)
// ---------------------------------------------------------------------

// Listener menunggu NOTIFY pada satu channel. Koneksi dibuat saat dibutuhkan dan dibuat ulang
// otomatis bila putus. Dipakai oleh satu goroutine saja.
type Listener struct {
	dsn, channel string
	conn         *pgx.Conn
}

func NewListener(dsn, channel string) *Listener { return &Listener{dsn: dsn, channel: channel} }

// Next menunggu satu NOTIFY dan mengembalikan payload-nya. Error bila ctx berakhir atau koneksi putus
// (pemanggil cukup memanggil Next lagi; koneksi disambung ulang).
func (l *Listener) Next(ctx context.Context) (string, error) {
	if l.conn == nil || l.conn.IsClosed() {
		conn, err := pgx.Connect(ctx, l.dsn)
		if err != nil {
			return "", err
		}
		if _, err := conn.Exec(ctx, "LISTEN "+pgx.Identifier{l.channel}.Sanitize()); err != nil {
			conn.Close(context.Background())
			return "", err
		}
		l.conn = conn
	}
	n, err := l.conn.WaitForNotification(ctx)
	if err != nil {
		if ctx.Err() == nil || l.conn.IsClosed() { // putus sungguhan → buang koneksi
			l.conn.Close(context.Background())
			l.conn = nil
		}
		return "", err
	}
	if n == nil {
		return "", errors.New("notifikasi kosong")
	}
	return n.Payload, nil
}

func (l *Listener) Close() {
	if l.conn != nil {
		l.conn.Close(context.Background())
	}
}

// LoadZones = zona peta yang berlaku + versi yang berakhir dalam 8 hari terakhir (untuk data terlambat).
func (r *Repository) LoadZones(ctx context.Context) ([]ZoneRow, error) {
	return collectRows(ctx, r, `
		SELECT zone_id, project_id, source_fid, category, name, priority, area_m2::float8,
		       ST_AsGeoJSON(geom, 7), valid_from, valid_to
		  FROM geo.v_iot_zones
		 WHERE valid_to IS NULL OR valid_to > now() - interval '8 days'`,
		func(row pgx.Rows, x *ZoneRow) error {
			return row.Scan(&x.ID, &x.ProjectID, &x.SourceFID, &x.Category, &x.Name, &x.Priority, &x.AreaM2,
				&x.GeoJSON, &x.ValidFrom, &x.ValidTo)
		})
}

// LoadCalibrations = titik kalibrasi semua equipment, urut per equipment & nilai sensor.
func (r *Repository) LoadCalibrations(ctx context.Context) ([]CalibrationRow, error) {
	return collectRows(ctx, r, `
		SELECT equipment_id, calibration_type, tank_capacity_liters::float8, sensor_value::float8, volume_liters::float8
		  FROM core.v_iot_fuel_calibration_points
		 ORDER BY equipment_id, sensor_value`,
		func(row pgx.Rows, x *CalibrationRow) error {
			return row.Scan(&x.EquipmentID, &x.Type, &x.CapacityLiters, &x.Sensor, &x.Liters)
		})
}

// LoadAlertRules = ambang efektif setiap project untuk tingkat project (zona NULL) dan setiap nama zona
// di peta yang berlaku — dihitung SEKALI oleh core.effective_alert_rules (satu sumber aturan, ADR 0008).
func (r *Repository) LoadAlertRules(ctx context.Context) ([]AlertRuleRow, error) {
	return collectRows(ctx, r, `
		SELECT p.project_id, z.name, e.alert_type, e.duration_minutes::float8, e.speed_limit_kph::float8,
		       e.fuel_drop_liters::float8, e.recipient_user_ids
		  FROM core.v_iot_projects p
		 CROSS JOIN LATERAL (SELECT NULL::text AS name
		                     UNION
		                     SELECT DISTINCT name FROM geo.v_iot_zones
		                      WHERE project_id = p.project_id AND valid_to IS NULL) z
		 CROSS JOIN LATERAL core.effective_alert_rules(p.project_id, z.name) e
		 WHERE p.is_active`,
		func(row pgx.Rows, x *AlertRuleRow) error {
			return row.Scan(&x.ProjectID, &x.ZoneName, &x.AlertType, &x.DurationMinutes, &x.SpeedLimitKph,
				&x.FuelDropLiters, &x.Recipients)
		})
}

// LoadRecipients = user aktif yang punya nomor HP (calon penerima WhatsApp).
func (r *Repository) LoadRecipients(ctx context.Context) ([]Recipient, error) {
	return collectRows(ctx, r, `SELECT user_id, name, phone_number FROM core.v_iot_alert_recipients`,
		func(row pgx.Rows, x *Recipient) error {
			return row.Scan(&x.UserID, &x.Name, &x.Phone)
		})
}
