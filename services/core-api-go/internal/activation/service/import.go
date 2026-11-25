package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/activation/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel aktivasi (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template + satu baris contoh (sama dengan template sistem lama).
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Equipment Code", "Status", "Description"},
		[]any{"11-08-2026", "TR-001", "Active", "Equipment aktif untuk operasional"}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal & unit" }

// ParseImportRow — Aktivasi: Date | Equipment Code | Status | Description.
// Unit boleh belum terdaftar di master equipment (seperti tiket timbangan): kodenya tetap disimpan.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	a := domain.ActivationRow{ProjectID: pd.Project.ID}

	date, err := excel.ParseDate(excel.Cell(row, 0))
	if err != nil {
		errs = append(errs, dailyinput.RowIssue{Column: "Date", Message: err.Error()})
	}
	a.Date = date

	if code := excel.Cell(row, 1); code == "" {
		errs = append(errs, dailyinput.RowIssue{Column: "Equipment Code", Message: "Equipment Code wajib diisi"})
	} else {
		var issues []dailyinput.RowIssue
		pd.SetEquipment(&a.EquipmentCode, &a.EquipmentID, code, &issues, "Equipment Code")
		errs = append(errs, issues...)
	}

	switch strings.ToLower(strings.Join(strings.Fields(excel.Cell(row, 2)), " ")) {
	case "active":
		a.Status = domain.ActivationActive
	case "non active", "inactive", "nonactive":
		a.Status = domain.ActivationInactive
	default:
		errs = append(errs, dailyinput.RowIssue{Column: "Status", Message: "Status harus Active atau Non Active"})
	}

	if d := excel.Cell(row, 3); d != "" {
		var issues []dailyinput.RowIssue
		a.Description = dailyinput.Optional(&d, "Description", 255, &issues)
		errs = append(errs, issues...)
	}
	return a.Date + "|" + a.EquipmentCode, a, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.ActivationRow](recs), by, dryRun)
}
