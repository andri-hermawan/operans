package service

import (
	"context"
	"strings"
	"unicode/utf8"

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

// ParseImportRow — Aktivasi: Date | Equipment Code | Status | Description — unit WAJIB terdaftar di master (B4).
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	var a domain.ActivationRow

	date, err := excel.ParseDate(excel.Cell(row, 0))
	if err != nil {
		errs = append(errs, dailyinput.RowIssue{Column: "Date", Message: err.Error()})
	}
	a.Date = date

	code := strings.ToUpper(excel.Cell(row, 1))
	if code == "" {
		errs = append(errs, dailyinput.RowIssue{Column: "Equipment Code", Message: "Equipment Code wajib diisi"})
	} else if id, ok := pd.Equipment[code]; ok {
		a.EquipmentID = id
	} else {
		errs = append(errs, dailyinput.RowIssue{Column: "Equipment Code", Message: "unit " + code + " belum terdaftar di master equipment"})
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
		if utf8.RuneCountInString(d) > 255 {
			errs = append(errs, dailyinput.RowIssue{Column: "Description", Message: "Description maks. 255 karakter"})
		}
		a.Description = &d
	}
	return a.Date + "|" + a.EquipmentID.String(), a, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.ActivationRow](recs), by, dryRun)
}
