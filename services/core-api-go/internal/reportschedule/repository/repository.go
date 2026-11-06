// Package repository (fitur reportschedule) = SQL ke tabel core.report_schedules & core.report_schedule_logs
// (zona waktu project & data penerima dibaca lewat JOIN ke core.projects & auth.users).
package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/reportschedule/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const scheduleSelect = `
	SELECT s.id, s.project_id, s.name, s.report_type, s.shift_code, s.date_offset, to_char(s.send_time, 'HH24:MI'),
	       s.recipient_user_ids, s.is_active, s.last_run_at, p.timezone, s.created_by, s.updated_by, s.updated_at,
	       coalesce((SELECT json_agg(json_build_object('user_id', u.id, 'name', u.name, 'email', u.email) ORDER BY u.name)
	                   FROM auth.users u WHERE u.id = ANY(s.recipient_user_ids) AND u.is_active), '[]')
	  FROM core.report_schedules s JOIN core.projects p ON p.id = s.project_id`

func scanSchedule(row pgx.Row) (*domain.ReportSchedule, error) {
	var s domain.ReportSchedule
	var recipients []byte
	err := row.Scan(&s.ID, &s.ProjectID, &s.Name, &s.ReportType, &s.ShiftCode, &s.DateOffset, &s.SendTime,
		&s.RecipientUserIDs, &s.IsActive, &s.LastRunAt, &s.Timezone, &s.CreatedBy, &s.UpdatedBy, &s.UpdatedAt, &recipients)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, json.Unmarshal(recipients, &s.Recipients)
}

// List: projectID nil = semua project; activeOnly untuk penjadwal.
func (r *Repository) List(ctx context.Context, projectID *uuid.UUID, activeOnly bool) ([]domain.ReportSchedule, error) {
	rows, err := r.pool.Query(ctx, scheduleSelect+`
		 WHERE ($1::uuid IS NULL OR s.project_id = $1) AND (NOT $2 OR (s.is_active AND p.is_active))
		 ORDER BY s.send_time, s.name`, projectID, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ReportSchedule{}
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.ReportSchedule, error) {
	return scanSchedule(r.pool.QueryRow(ctx, scheduleSelect+` WHERE s.id = $1`, id))
}

func (r *Repository) Insert(ctx context.Context, s domain.ReportSchedule, by uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.report_schedules (project_id, name, report_type, shift_code, date_offset, send_time,
		                                   recipient_user_ids, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6::time, $7, $8, $9, $9) RETURNING id`,
		s.ProjectID, s.Name, s.ReportType, s.ShiftCode, s.DateOffset, s.SendTime, s.RecipientUserIDs, s.IsActive, by).Scan(&id)
	return id, err
}

func (r *Repository) Update(ctx context.Context, s domain.ReportSchedule, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.report_schedules
		   SET name = $2, report_type = $3, shift_code = $4, date_offset = $5, send_time = $6::time,
		       recipient_user_ids = $7, is_active = $8, updated_by = $9
		 WHERE id = $1`,
		s.ID, s.Name, s.ReportType, s.ShiftCode, s.DateOffset, s.SendTime, s.RecipientUserIDs, s.IsActive, by)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.report_schedules WHERE id = $1`, id)
	return err
}

// HasScheduledRun: sudah ada pengiriman otomatis untuk jadwal ini pada tanggal kirim tsb (sukses atau gagal).
func (r *Repository) HasScheduledRun(ctx context.Context, id uuid.UUID, runDate string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM core.report_schedule_logs
		 WHERE schedule_id = $1 AND run_date = $2::date AND trigger = 'schedule')`, id, runDate).Scan(&ok)
	return ok, err
}

func (r *Repository) InsertLog(ctx context.Context, id uuid.UUID, l domain.ReportScheduleLog) error {
	_, err := r.pool.Exec(ctx, `
		WITH log AS (
		  INSERT INTO core.report_schedule_logs (schedule_id, run_date, report_date, trigger, status, recipients, error_message)
		  VALUES ($1, $2::date, $3::date, $4, $5, $6, $7))
		UPDATE core.report_schedules SET last_run_at = now() WHERE id = $1`,
		id, l.RunDate, l.ReportDate, l.Trigger, l.Status, l.Recipients, l.ErrorMessage)
	return err
}

func (r *Repository) ListLogs(ctx context.Context, id uuid.UUID) ([]domain.ReportScheduleLog, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_date::text, report_date::text, trigger, status, recipients, error_message, created_at
		  FROM core.report_schedule_logs WHERE schedule_id = $1 ORDER BY created_at DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ReportScheduleLog{}
	for rows.Next() {
		var l domain.ReportScheduleLog
		if err := rows.Scan(&l.ID, &l.RunDate, &l.ReportDate, &l.Trigger, &l.Status, &l.Recipients, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
