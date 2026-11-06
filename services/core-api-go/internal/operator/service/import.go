package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/operator/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel operator (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template + satu baris contoh (sama dengan template sistem lama).
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Tanggal", "Shift", "Asset ID", "Operator Name", "Description"},
		[]any{"11-08-2026", "Shift 1", "EQ-001", "John Doe", ""}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift & unit" }

// ParseImportRow — Operator: Tanggal | Shift | Asset ID | Operator Name | Description
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	o := domain.OperatorAssignment{ProjectID: pd.Project.ID}

	date, err := excel.ParseDate(excel.Cell(row, 0))
	if err != nil {
		errs = append(errs, dailyinput.RowIssue{Column: "Tanggal", Message: err.Error()})
	}
	o.OperationalDate = date

	in := OperatorInput{
		Shift:         dailyinput.PtrStr(excel.Cell(row, 1)),
		EquipmentCode: dailyinput.PtrStr(excel.Cell(row, 2)),
		OperatorName:  dailyinput.PtrStr(excel.Cell(row, 3)),
		Description:   dailyinput.PtrStr(excel.Cell(row, 4)),
	}
	// full=false: tanggal sudah diurai di atas; kolom lain tetap wajib karena pointer-nya selalu diisi
	errs = append(errs, dailyinput.RenameColumns(applyOperator(pd, &o, in, false), map[string]string{"shift": "Shift",
		"equipment_code": "Asset ID", "operator_name": "Operator Name", "description": "Description"})...)
	if o.Description != nil && *o.Description == "" {
		o.Description = nil
	}
	return o.OperationalDate + "|" + o.ShiftID.String() + "|" + o.EquipmentCode, o, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.OperatorAssignment](recs), by, dryRun)
}
