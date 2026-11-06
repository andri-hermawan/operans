// Package repository (fitur fuelcalibration) = semua SQL ke tabel core.fuel_calibrations &
// core.fuel_calibration_points (kode unit & project dibaca lewat JOIN ke core.equipment).
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/fuelcalibration/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const fuelCalibrationSQL = `
SELECT c.equipment_id, e.equipment_code, e.project_id, c.calibration_type, c.tank_capacity_liters::float8,
       c.max_sensor_value::float8, c.is_linear,
       (SELECT count(*) FROM core.fuel_calibration_points p WHERE p.equipment_id = c.equipment_id), c.updated_at
  FROM core.fuel_calibrations c
  JOIN core.equipment e ON e.id = c.equipment_id`

func scanFuelCalibration(row pgx.Row) (*domain.FuelCalibration, error) {
	var c domain.FuelCalibration
	err := row.Scan(&c.EquipmentID, &c.EquipmentCode, &c.ProjectID, &c.CalibrationType, &c.TankCapacityLiters,
		&c.MaxSensorValue, &c.IsLinear, &c.PointCount, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List = ringkasan kalibrasi semua equipment project (tanpa titik).
func (r *Repository) List(ctx context.Context, projectID uuid.UUID) ([]domain.FuelCalibration, error) {
	rows, err := r.pool.Query(ctx, fuelCalibrationSQL+` WHERE e.project_id = $1 ORDER BY e.equipment_code`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FuelCalibration{}
	for rows.Next() {
		c, err := scanFuelCalibration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// Get = kalibrasi + semua titik (urut nilai sensor). nil bila belum ada.
func (r *Repository) Get(ctx context.Context, equipmentID uuid.UUID) (*domain.FuelCalibration, error) {
	c, err := scanFuelCalibration(r.pool.QueryRow(ctx, fuelCalibrationSQL+` WHERE c.equipment_id = $1`, equipmentID))
	if err != nil || c == nil {
		return c, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT sensor_value::float8, volume_liters::float8 FROM core.fuel_calibration_points
		 WHERE equipment_id = $1 ORDER BY sensor_value`, equipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c.Points = []domain.CalibrationPoint{}
	for rows.Next() {
		var p domain.CalibrationPoint
		if err := rows.Scan(&p.SensorValue, &p.VolumeLiters); err != nil {
			return nil, err
		}
		c.Points = append(c.Points, p)
	}
	return c, rows.Err()
}

// Save mengganti kalibrasi equipment (kepala + semua titik) dalam satu transaksi.
// created=true bila sebelumnya belum ada.
func (r *Repository) Save(ctx context.Context, c domain.FuelCalibration, by uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var created bool
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.fuel_calibrations (equipment_id, calibration_type, tank_capacity_liters, max_sensor_value,
		                                    is_linear, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (equipment_id) DO UPDATE SET calibration_type = EXCLUDED.calibration_type,
		       tank_capacity_liters = EXCLUDED.tank_capacity_liters, max_sensor_value = EXCLUDED.max_sensor_value,
		       is_linear = EXCLUDED.is_linear, updated_by = EXCLUDED.updated_by
		RETURNING (xmax = 0)`,
		c.EquipmentID, c.CalibrationType, c.TankCapacityLiters, c.MaxSensorValue, c.IsLinear, by).Scan(&created); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.fuel_calibration_points WHERE equipment_id = $1`, c.EquipmentID); err != nil {
		return false, err
	}
	rows := make([][]any, len(c.Points))
	for i, p := range c.Points {
		rows[i] = []any{c.EquipmentID, p.SensorValue, p.VolumeLiters}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"core", "fuel_calibration_points"},
		[]string{"equipment_id", "sensor_value", "volume_liters"}, pgx.CopyFromRows(rows)); err != nil {
		return false, err
	}
	return created, tx.Commit(ctx)
}

// Delete menghapus kalibrasi (titik ikut terhapus). false bila tidak ada.
func (r *Repository) Delete(ctx context.Context, equipmentID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM core.fuel_calibrations WHERE equipment_id = $1`, equipmentID)
	return tag.RowsAffected() > 0, err
}
