package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/rainslippery/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel Rain & Slippery (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "Location", "StartRain", "StopRain", "TotalRain", "StartWaitSlippery", "StopWaitSlippery",
			"TotalWaitSlippery", "StartSlippery", "StopSlippery", "TotalSlippery", "RainFall"},
		[]any{"01/05/2025", "Shift 1", "Pit 1", "09:00", "10:30", 1.5, "10:30", "11:00", 0.5, "11:00", "12:00", 1, 12}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift & lokasi" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var rainSlipperyColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "location": "Location",
	"rain_start": "StartRain", "rain_stop": "StopRain", "rain_hours": "TotalRain",
	"wait_slippery_start": "StartWaitSlippery", "wait_slippery_stop": "StopWaitSlippery", "wait_slippery_hours": "TotalWaitSlippery",
	"slippery_start": "StartSlippery", "slippery_stop": "StopSlippery", "slippery_hours": "TotalSlippery", "rainfall_mm": "RainFall"}

// ParseImportRow — Rain & Slippery: kolom sesuai ImportColumns (13 kolom).
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	num := func(i int, col string) *float64 { return dailyinput.NumCell(row, i, col, excel.ParseNumber, &errs) }
	in := RainSlipperyInput{
		OperationalDate: cell(0), Shift: cell(1), Location: cell(2),
		RainStart: cell(3), RainStop: cell(4), RainHours: num(5, "rain_hours"),
		WaitSlipperyStart: cell(6), WaitSlipperyStop: cell(7), WaitSlipperyHours: num(8, "wait_slippery_hours"),
		SlipperyStart: cell(9), SlipperyStop: cell(10), SlipperyHours: num(11, "slippery_hours"),
		RainfallMm: num(12, "rainfall_mm"),
	}
	t := domain.RainSlipperyLog{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyRainSlippery(pd, &t, in)...), rainSlipperyColumns)
	loc := ""
	if t.Location != nil {
		loc = *t.Location
	}
	return t.OperationalDate + "|" + t.ShiftCode + "|" + loc, t, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.RainSlipperyLog](recs), by, dryRun)
}
