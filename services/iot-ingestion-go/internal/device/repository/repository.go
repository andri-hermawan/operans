// Package repository = semua SQL fitur Device (tabel iot.devices).
// DeviceLastEvent hanya MEMBACA iot.telemetry_events (tabel itu ditulis oleh fitur ingest).
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/shared/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// FindDeviceByIMEI dipakai listener TCP: Teltonika memperkenalkan diri dengan IMEI.
// Mengembalikan (nil, nil) bila IMEI tidak terdaftar atau alat dinonaktifkan.
func (r *Repository) FindDeviceByIMEI(ctx context.Context, imei string) (*domain.Device, error) {
	var d domain.Device
	err := r.pool.QueryRow(ctx,
		`SELECT id, imei, equipment_id, project_id
		   FROM iot.devices
		  WHERE imei = $1 AND is_active`,
		imei).Scan(&d.ID, &d.IMEI, &d.EquipmentID, &d.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// FindDeviceByKeyHash dipakai endpoint HTTP: alat/simulator mengirim API key, kita cari hash-nya.
// Mengembalikan (nil, nil) bila key tidak dikenal atau alat dinonaktifkan.
func (r *Repository) FindDeviceByKeyHash(ctx context.Context, hash string) (*domain.Device, error) {
	var d domain.Device
	err := r.pool.QueryRow(ctx,
		`SELECT id, imei, equipment_id, project_id
		   FROM iot.devices
		  WHERE api_key_hash = $1 AND is_active`,
		hash).Scan(&d.ID, &d.IMEI, &d.EquipmentID, &d.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

const deviceRowCols = `d.id, d.imei, d.name, d.project_id, d.equipment_id, e.equipment_code, d.model, d.fuel_source, d.sim_number,
    d.provider_name, d.api_key_hash IS NOT NULL, d.is_active, l.event_time, d.updated_at`

const deviceRowFrom = ` FROM iot.devices d
    LEFT JOIN core.v_iot_equipment e ON e.equipment_id = d.equipment_id
    LEFT JOIN iot.equipment_live_states l ON l.equipment_id = d.equipment_id`

func scanDeviceRow(row pgx.Row) (*domain.DeviceRow, error) {
	var d domain.DeviceRow
	err := row.Scan(&d.ID, &d.IMEI, &d.Name, &d.ProjectID, &d.EquipmentID, &d.EquipmentCode, &d.Model, &d.FuelSource,
		&d.SimNumber, &d.ProviderName, &d.HasAPIKey, &d.IsActive, &d.LastEventAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &d, err
}

// ListDevices: projectID nil = semua (superadmin). Alat tanpa project hanya terlihat oleh superadmin.
func (r *Repository) ListDevices(ctx context.Context, projectID *uuid.UUID) ([]domain.DeviceRow, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+deviceRowCols+deviceRowFrom+`
		 WHERE $1::uuid IS NULL OR d.project_id = $1 ORDER BY d.imei`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DeviceRow{}
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repository) GetDevice(ctx context.Context, id uuid.UUID) (*domain.DeviceRow, error) {
	return scanDeviceRow(r.pool.QueryRow(ctx, `SELECT `+deviceRowCols+deviceRowFrom+` WHERE d.id = $1`, id))
}

// EquipmentProject = project pemilik equipment (nil bila tidak ada).
func (r *Repository) EquipmentProject(ctx context.Context, equipmentID uuid.UUID) (*uuid.UUID, error) {
	var pid uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT project_id FROM core.v_iot_equipment WHERE equipment_id = $1`, equipmentID).Scan(&pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &pid, err
}

// DeviceOnEquipment = alat AKTIF lain yang sudah terpasang di equipment ini (satu unit satu alat aktif).
func (r *Repository) DeviceOnEquipment(ctx context.Context, equipmentID uuid.UUID, except *uuid.UUID) (string, error) {
	var imei string
	err := r.pool.QueryRow(ctx, `SELECT imei FROM iot.devices WHERE equipment_id = $1 AND is_active AND ($2::uuid IS NULL OR id <> $2) LIMIT 1`,
		equipmentID, except).Scan(&imei)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return imei, err
}

func (r *Repository) InsertDevice(ctx context.Context, in domain.DeviceInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO iot.devices (imei, name, project_id, equipment_id, model, fuel_source, sim_number, provider_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, true)) RETURNING id`,
		in.IMEI, in.Name, in.ProjectID, in.EquipmentID, in.Model, in.FuelSource, in.SimNumber, in.ProviderName, in.IsActive).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, domain.ErrIMEIExists
	}
	return id, err
}

// UpdateDevice: IMEI tidak bisa diubah (identitas alat Teltonika).
func (r *Repository) UpdateDevice(ctx context.Context, id uuid.UUID, in domain.DeviceInput) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE iot.devices
		   SET name = COALESCE($2, name),
		       equipment_id = CASE WHEN $3 THEN NULL ELSE COALESCE($4, equipment_id) END,
		       model = COALESCE($5, model), fuel_source = CASE WHEN $6::text = '' THEN NULL ELSE COALESCE($6, fuel_source) END,
		       sim_number = COALESCE($7, sim_number), provider_name = COALESCE($8, provider_name),
		       is_active = COALESCE($9, is_active), updated_at = now()
		 WHERE id = $1`,
		id, in.Name, in.ClearUnit, in.EquipmentID, in.Model, in.FuelSource, in.SimNumber, in.ProviderName, in.IsActive)
	return err
}

// SetDeviceKeyHash menyimpan hash API key baru (key lama otomatis tidak berlaku).
func (r *Repository) SetDeviceKeyHash(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE iot.devices SET api_key_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	return err
}

// DeviceLastEvent = waktu event terbaru sebuah alat & kapan event itu diterima server.
// (nil, nil, nil) bila alat belum pernah mengirim data (atau datanya bukan milik project ini).
func (r *Repository) DeviceLastEvent(ctx context.Context, projectID *uuid.UUID, deviceID uuid.UUID) (eventTime, receivedAt *time.Time, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT event_time, created_at
		   FROM iot.telemetry_events
		  WHERE device_id = $1
		    AND ($2::uuid IS NULL OR project_id = $2)
		  ORDER BY event_time DESC
		  LIMIT 1`,
		deviceID, projectID).Scan(&eventTime, &receivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return eventTime, receivedAt, err
}
