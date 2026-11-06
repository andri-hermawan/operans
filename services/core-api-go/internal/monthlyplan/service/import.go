package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"simha/core-api/internal/monthlyplan/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/shared/plan"
)

// ---------------------------------------------------------------------
// Impor Excel Plan Monthly (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// dayTitles = judul kolom tanggal di template: T01..T31 (sama dengan tabel lama).
var dayTitles = func() []string {
	t := make([]string, 31)
	for i := range t {
		t[i] = fmt.Sprintf("T%02d", i+1)
	}
	return t
}()

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	headers = append([]string{"Mitra", "Year", "Month", "Description", "Name", "Uom", "Total"}, dayTitles...)
	example = []any{"TBBE", 2025, 6, "Volume", "Waste Removal", "BCM", 147409.45}
	for _, v := range []float64{5075.38, 5075.38, 5075.38, 5075.38, 5075.38, 4485.98, 4972.25, 4972.25, 4972.25, 4917.19, 4972.25, 4972.25, 4394.82,
		4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4394.82, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4394.82,
		4972.25, 4972.25, 4972.25, 0} {
		example = append(example, v)
	}
	return headers, example
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "mitra, tahun, bulan, deskripsi & nama" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var monthlyColumns = func() map[string]string {
	m := map[string]string{"partner": "Mitra", "plan_year": "Year", "plan_month": "Month", "description": "Description",
		"plan_name": "Name", "uom": "Uom", "total": "Total"}
	for i, c := range domain.DayColumns {
		m[c] = dayTitles[i]
	}
	return m
}()

// ParseImportRow — Plan Monthly: Mitra | Year | Month | Description | Name | Uom | Total | T01 … T31.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := MonthlyPlanInput{
		Partner: cell(0), Description: cell(3), PlanName: cell(4), Uom: cell(5),
		PlanYear:  plan.Whole(dailyinput.NumCell(row, 1, "plan_year", excel.ParseNumber, &errs), "plan_year", &errs),
		PlanMonth: plan.Whole(dailyinput.NumCell(row, 2, "plan_month", excel.ParseNumber, &errs), "plan_month", &errs),
		Total:     dailyinput.NumCell(row, 6, "total", excel.ParseNumber, &errs),
		Days:      make([]*float64, len(domain.DayColumns)),
	}
	for i, c := range domain.DayColumns {
		in.Days[i] = dailyinput.NumCell(row, 7+i, c, excel.ParseNumber, &errs)
	}
	p := domain.MonthlyPlan{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyMonthly(&p, in)...), monthlyColumns)
	return fmt.Sprintf("%s|%d|%d|%s|%s", p.Partner, p.PlanYear, p.PlanMonth, p.Description, p.PlanName), p, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.MonthlyPlan](recs), by, dryRun)
}
