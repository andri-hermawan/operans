// Package repository (fitur alertrule) = SQL ke tabel core.alert_rules & core.alert_rule_recipients,
// plus fungsi database core.effective_alert_rules (ambang yang berlaku).
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/alertrule/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const alertRuleSQL = `
SELECT r.id, r.project_id, r.alert_type, t.name, r.zone_name, r.duration_minutes::float8, r.speed_limit_kph::float8,
       r.fuel_drop_liters::float8, r.description, r.is_active, r.updated_at,
       coalesce((SELECT json_agg(json_build_object('user_id', u.id, 'name', u.name, 'phone_number', u.phone_number)
                                 ORDER BY u.name)
                   FROM core.alert_rule_recipients x JOIN auth.users u ON u.id = x.user_id
                  WHERE x.alert_rule_id = r.id), '[]')
  FROM core.alert_rules r
  JOIN core.alert_types t ON t.code = r.alert_type`

func scanAlertRule(row pgx.Row) (*domain.AlertRule, error) {
	var a domain.AlertRule
	err := row.Scan(&a.ID, &a.ProjectID, &a.AlertType, &a.AlertTypeName, &a.ZoneName, &a.DurationMinutes,
		&a.SpeedLimitKph, &a.FuelDropLiters, &a.Description, &a.IsActive, &a.UpdatedAt, &a.Recipients)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) List(ctx context.Context, projectID uuid.UUID) ([]domain.AlertRule, error) {
	rows, err := r.pool.Query(ctx, alertRuleSQL+`
		 WHERE r.project_id = $1 ORDER BY t.sort_order, r.zone_name NULLS FIRST`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AlertRule{}
	for rows.Next() {
		a, err := scanAlertRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	return scanAlertRule(r.pool.QueryRow(ctx, alertRuleSQL+` WHERE r.id = $1`, id))
}

// Save menambah (ID kosong) atau mengganti seluruh isi aturan + penerimanya dalam satu transaksi.
func (r *Repository) Save(ctx context.Context, a domain.AlertRule, by uuid.UUID) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	id := a.ID
	if id == uuid.Nil {
		err = tx.QueryRow(ctx, `
			INSERT INTO core.alert_rules (project_id, alert_type, zone_name, duration_minutes, speed_limit_kph,
			                              fuel_drop_liters, description, is_active, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9) RETURNING id`,
			a.ProjectID, a.AlertType, a.ZoneName, a.DurationMinutes, a.SpeedLimitKph, a.FuelDropLiters,
			a.Description, a.IsActive, by).Scan(&id)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE core.alert_rules SET alert_type = $2, zone_name = $3, duration_minutes = $4, speed_limit_kph = $5,
			       fuel_drop_liters = $6, description = $7, is_active = $8, updated_by = $9
			 WHERE id = $1`,
			id, a.AlertType, a.ZoneName, a.DurationMinutes, a.SpeedLimitKph, a.FuelDropLiters, a.Description, a.IsActive, by)
	}
	if database.IsUniqueViolation(err) {
		return uuid.Nil, fmt.Errorf("%w: aturan untuk jenis alert & zona ini", apperror.ErrDuplicate)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.alert_rule_recipients WHERE alert_rule_id = $1`, id); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.alert_rule_recipients (alert_rule_id, user_id, created_by)
		SELECT $1, unnest($2::uuid[]), $3`, id, a.RecipientUserIDs, by); err != nil {
		return uuid.Nil, err
	}
	return id, tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.alert_rules WHERE id = $1`, id)
	return err
}

// Effective = ambang berlaku per jenis (fungsi core.effective_alert_rules) + data penerima.
func (r *Repository) Effective(ctx context.Context, projectID uuid.UUID, zoneName string) ([]domain.EffectiveAlertRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.alert_type, t.name, e.duration_minutes::float8, e.speed_limit_kph::float8, e.fuel_drop_liters::float8,
		       coalesce((SELECT json_agg(json_build_object('user_id', u.id, 'name', u.name, 'phone_number', u.phone_number)
		                                 ORDER BY u.name)
		                   FROM auth.users u WHERE u.id = ANY (e.recipient_user_ids)), '[]')
		  FROM core.effective_alert_rules($1, $2) e
		  JOIN core.alert_types t ON t.code = e.alert_type
		 ORDER BY t.sort_order`, projectID, zoneName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.EffectiveAlertRule{}
	for rows.Next() {
		var e domain.EffectiveAlertRule
		if err := rows.Scan(&e.AlertType, &e.Name, &e.DurationMinutes, &e.SpeedLimitKph, &e.FuelDropLiters, &e.Recipients); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ProjectMembers = anggota project yang AKTIF di antara ids → data penerima (untuk validasi penerima alert).
//
// PENGECUALIAN (sengaja): query BACA-SAJA ke tabel milik fitur user (auth.users, auth.project_members),
// sama seperti JOIN penerima di alertRuleSQL. Hasilnya langsung berbentuk AlertRecipient milik fitur ini.
func (r *Repository) ProjectMembers(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.AlertRecipient, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.name, u.phone_number
		  FROM auth.users u JOIN auth.project_members m ON m.user_id = u.id AND m.project_id = $1
		 WHERE u.is_active AND u.id = ANY ($2)`, projectID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]domain.AlertRecipient{}
	for rows.Next() {
		var a domain.AlertRecipient
		if err := rows.Scan(&a.UserID, &a.Name, &a.PhoneNumber); err != nil {
			return nil, err
		}
		out[a.UserID] = a
	}
	return out, rows.Err()
}
