// Package repository (fitur report) = SQL rekap laporan harian (per unit per tanggal).
//
// PENGECUALIAN (sengaja): laporan hanya MEMBACA & menjumlahkan tabel milik fitur weighbridge
// (core.weighbridge_tickets) dan breakdown (core.equipment_breakdowns) dengan GROUP BY di database —
// jauh lebih cepat daripada mengambil semua baris lewat service fitur tersebut.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/report/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// WeighbridgeDaily: per unit per tanggal → jumlah tiket & total net (ritase, payload, produksi; rumus sistem lama).
func (r *Repository) WeighbridgeDaily(ctx context.Context, f domain.ReportFilter) ([]domain.DailyCell, error) {
	return r.dailyCells(ctx, `
		SELECT equipment_code, operational_date::text, count(*), coalesce(sum(net_kg), 0), 0
		  FROM core.weighbridge_tickets
		 WHERE project_id = $1 AND operational_date BETWEEN $2::date AND $3::date
		   AND ($4::uuid IS NULL OR shift_id = $4) AND (cardinality($5::text[]) = 0 OR equipment_code = ANY($5))
		 GROUP BY 1, 2 ORDER BY 1, 2`, f)
}

// ExecutiveHauling: per customer → net kg pada tanggal laporan, 1 bulan s.d. tanggal itu (MTD), dan 1 Januari s.d. tanggal itu (YTD).
func (r *Repository) ExecutiveHauling(ctx context.Context, projectID uuid.UUID, date string) ([]domain.ExecutiveRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT coalesce(nullif(upper(btrim(customer)), ''), '-'),
		       coalesce(sum(net_kg) FILTER (WHERE operational_date = $2::date), 0),
		       coalesce(sum(net_kg) FILTER (WHERE operational_date >= date_trunc('month', $2::date)), 0),
		       coalesce(sum(net_kg), 0)
		  FROM core.weighbridge_tickets
		 WHERE project_id = $1 AND operational_date BETWEEN date_trunc('year', $2::date)::date AND $2::date
		 GROUP BY 1 ORDER BY 1`, projectID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ExecutiveRow{}
	for rows.Next() {
		var e domain.ExecutiveRow
		if err := rows.Scan(&e.Customer, &e.DailyKg, &e.MtdKg, &e.YtdKg); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ExecutiveEmplacement: per partner dari core.stock_emplacements project itu — total_tls_out (harian/MTD/YTD) dan remaining_cargo pada tanggal laporan.
func (r *Repository) ExecutiveEmplacement(ctx context.Context, projectID uuid.UUID, date string) ([]domain.ExecutiveEmplacementRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT upper(btrim(partner)),
		       coalesce(sum(total_tls_out) FILTER (WHERE operational_date = $1::date), 0),
		       coalesce(sum(total_tls_out) FILTER (WHERE operational_date >= date_trunc('month', $1::date)), 0),
		       coalesce(sum(total_tls_out), 0),
		       coalesce(sum(remaining_cargo) FILTER (WHERE operational_date = $1::date), 0)
		  FROM core.stock_emplacements
		 WHERE project_id = $2 AND operational_date BETWEEN date_trunc('year', $1::date)::date AND $1::date
		 GROUP BY 1 ORDER BY 1`, date, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ExecutiveEmplacementRow{}
	for rows.Next() {
		var e domain.ExecutiveEmplacementRow
		if err := rows.Scan(&e.Partner, &e.DailyProd, &e.MtdProd, &e.YtdProd, &e.Inventory); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// BreakdownDaily: per unit per tanggal → jumlah jam breakdown (lama: SUM(duration) per tanggal & unit).
func (r *Repository) BreakdownDaily(ctx context.Context, f domain.ReportFilter) ([]domain.DailyCell, error) {
	return r.dailyCells(ctx, `
		SELECT equipment_code, operational_date::text, count(*), 0, coalesce(sum(duration_hours), 0)
		  FROM core.equipment_breakdowns
		 WHERE project_id = $1 AND operational_date BETWEEN $2::date AND $3::date
		   AND ($4::uuid IS NULL OR shift_id = $4) AND (cardinality($5::text[]) = 0 OR equipment_code = ANY($5))
		 GROUP BY 1, 2 ORDER BY 1, 2`, f)
}

func (r *Repository) dailyCells(ctx context.Context, sql string, f domain.ReportFilter) ([]domain.DailyCell, error) {
	codes := f.Codes
	if codes == nil {
		codes = []string{}
	}
	rows, err := r.pool.Query(ctx, sql, f.ProjectID, f.From, f.To, f.ShiftID, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DailyCell{}
	for rows.Next() {
		var c domain.DailyCell
		if err := rows.Scan(&c.EquipmentCode, &c.Date, &c.Count, &c.SumKg, &c.SumHours); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
