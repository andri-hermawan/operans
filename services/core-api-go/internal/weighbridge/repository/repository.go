// Package repository (fitur weighbridge) = semua SQL ke tabel core.weighbridge_tickets
// (kode & nama shift dibaca lewat JOIN ke core.shifts).
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/weighbridge/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const weighbridgeSQL = `
SELECT w.id, w.project_id, w.operational_date::text, w.shift_id, s.shift_code, s.shift_name,
       w.ticket_no, w.equipment_code, w.equipment_id, w.product,
       w.gross_kg::float8, w.tare_kg::float8, w.net_kg::float8,
       w.recipient, w.customer, w.transporter, w.gross_at, w.tare_at,
       w.gross_operator, w.tare_operator, w.description, w.location, w.updated_at
  FROM core.weighbridge_tickets w
  JOIN core.shifts s ON s.id = w.shift_id`

func scanWeighbridge(row pgx.Row) (*domain.WeighbridgeTicket, error) {
	var w domain.WeighbridgeTicket
	err := row.Scan(&w.ID, &w.ProjectID, &w.OperationalDate, &w.ShiftID, &w.ShiftCode, &w.ShiftName,
		&w.TicketNo, &w.EquipmentCode, &w.EquipmentID, &w.Product, &w.GrossKg, &w.TareKg, &w.NetKg,
		&w.Recipient, &w.Customer, &w.Transporter, &w.GrossAt, &w.TareAt,
		&w.GrossOperator, &w.TareOperator, &w.Description, &w.Location, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.WeighbridgeTicket, error) {
	rows, err := r.pool.Query(ctx, weighbridgeSQL+`
		 WHERE ($1::uuid IS NULL OR w.project_id = $1)
		   AND ($2::text = '' OR w.operational_date >= $2::text::date)
		   AND ($3::text = '' OR w.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR w.shift_id = $4)
		   AND ($5::text = '' OR w.equipment_code = $5)
		 ORDER BY w.operational_date DESC, s.sequence, w.gross_at NULLS LAST, w.ticket_no
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.WeighbridgeTicket{}
	for rows.Next() {
		w, err := scanWeighbridge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.WeighbridgeTicket, error) {
	return scanWeighbridge(r.pool.QueryRow(ctx, weighbridgeSQL+` WHERE w.id = $1`, id))
}

// Kunci "data sama ditimpa" (ADR 0007): tanggal + nomor tiket.
const upsertWeighbridgeSQL = `
INSERT INTO core.weighbridge_tickets (project_id, operational_date, shift_id, ticket_no, equipment_code, equipment_id,
       product, gross_kg, tare_kg, net_kg, recipient, customer, transporter, gross_at, tare_at,
       gross_operator, tare_operator, description, location, created_by, updated_by)
VALUES ($1, $2::text::date, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $20)
ON CONFLICT ON CONSTRAINT uq_weighbridge_tickets_key
DO UPDATE SET shift_id = EXCLUDED.shift_id, equipment_code = EXCLUDED.equipment_code, equipment_id = EXCLUDED.equipment_id,
              product = EXCLUDED.product, gross_kg = EXCLUDED.gross_kg, tare_kg = EXCLUDED.tare_kg, net_kg = EXCLUDED.net_kg,
              recipient = EXCLUDED.recipient, customer = EXCLUDED.customer, transporter = EXCLUDED.transporter,
              gross_at = EXCLUDED.gross_at, tare_at = EXCLUDED.tare_at, gross_operator = EXCLUDED.gross_operator,
              tare_operator = EXCLUDED.tare_operator, description = EXCLUDED.description, location = EXCLUDED.location,
              updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`

func weighbridgeArgs(w domain.WeighbridgeTicket, by uuid.UUID) []any {
	return []any{w.ProjectID, w.OperationalDate, w.ShiftID, w.TicketNo, w.EquipmentCode, w.EquipmentID,
		w.Product, w.GrossKg, w.TareKg, w.NetKg, w.Recipient, w.Customer, w.Transporter, w.GrossAt, w.TareAt,
		w.GrossOperator, w.TareOperator, w.Description, w.Location, by}
}

func (r *Repository) Upsert(ctx context.Context, w domain.WeighbridgeTicket, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertWeighbridgeSQL, weighbridgeArgs(w, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, w domain.WeighbridgeTicket, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.weighbridge_tickets
		   SET operational_date = $2::text::date, shift_id = $3, ticket_no = $4, equipment_code = $5, equipment_id = $6,
		       product = $7, gross_kg = $8, tare_kg = $9, net_kg = $10, recipient = $11, customer = $12,
		       transporter = $13, gross_at = $14, tare_at = $15, gross_operator = $16, tare_operator = $17,
		       description = $18, location = $19, updated_by = $20
		 WHERE id = $1`,
		w.ID, w.OperationalDate, w.ShiftID, w.TicketNo, w.EquipmentCode, w.EquipmentID, w.Product,
		w.GrossKg, w.TareKg, w.NetKg, w.Recipient, w.Customer, w.Transporter, w.GrossAt, w.TareAt,
		w.GrossOperator, w.TareOperator, w.Description, w.Location, by)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: nomor tiket pada tanggal itu sudah dipakai", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.weighbridge_tickets WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.WeighbridgeTicket, by uuid.UUID, dryRun bool) (int, int, error) {
	args := make([][]any, len(rows))
	for i, w := range rows {
		args[i] = weighbridgeArgs(w, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertWeighbridgeSQL, args, dryRun)
}
