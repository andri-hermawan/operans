package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/shared/plan"
	"simha/core-api/internal/yearlyplan/domain"
)

// ---------------------------------------------------------------------
// Impor Excel Plan Yearly (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

var monthTitles = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	headers = append([]string{"Mitra", "Year", "Location", "Description", "Name", "Total"}, monthTitles...)
	example = []any{"TBBE", 2025, "PIT A2", "Volume", "Waste Removal", 1325498.08,
		180055.41, 132485.47, 45757.93, 176691.46, 213976.63, 147409.45, 104064.95, 102978.01, 111148.41, 83034.93, 27895.43, 0}
	return headers, example
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "mitra, tahun, lokasi, deskripsi & nama" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var yearlyColumns = func() map[string]string {
	m := map[string]string{"partner": "Mitra", "plan_year": "Year", "location": "Location", "description": "Description",
		"plan_name": "Name", "total": "Total"}
	for i, c := range domain.MonthColumns {
		m[c] = monthTitles[i]
	}
	return m
}()

// ParseImportRow — Plan Yearly: Mitra | Year | Location | Description | Name | Total | Jan … Dec.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := YearlyPlanInput{
		Partner: cell(0), Location: cell(2), Description: cell(3), PlanName: cell(4),
		PlanYear: plan.Whole(dailyinput.NumCell(row, 1, "plan_year", excel.ParseNumber, &errs), "plan_year", &errs),
		Total:    dailyinput.NumCell(row, 5, "total", excel.ParseNumber, &errs),
		Months:   make([]*float64, len(domain.MonthColumns)),
	}
	for i, c := range domain.MonthColumns {
		in.Months[i] = dailyinput.NumCell(row, 6+i, c, excel.ParseNumber, &errs)
	}
	p := domain.YearlyPlan{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyYearly(&p, in)...), yearlyColumns)
	return fmt.Sprintf("%s|%d|%s|%s|%s", p.Partner, p.PlanYear, p.Location, p.Description, p.PlanName), p, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.YearlyPlan](recs), by, dryRun)
}
