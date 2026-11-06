package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/timesheet/domain"
)

// ---------------------------------------------------------------------
// Impor Excel Time Sheet (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "CnUnit", "NikOperator", "OperatorName", "HmStart", "HmEnd", "Total", "Remark"},
		[]any{"01/05/2025", "Shift 1", "DZ20034", "", "", 0, 0, 0, ""}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift & unit" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var timeSheetColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "equipment_code": "CnUnit",
	"operator_nik": "NikOperator", "operator_name": "OperatorName", "hm_start": "HmStart", "hm_end": "HmEnd",
	"hm_total": "Total", "remark": "Remark"}

// ParseImportRow — Time Sheet: Date | Shift | CnUnit | NikOperator | OperatorName | HmStart | HmEnd | Total | Remark.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := TimeSheetInput{
		OperationalDate: cell(0), Shift: cell(1), EquipmentCode: cell(2), OperatorNik: cell(3), OperatorName: cell(4), Remark: cell(8),
		HmStart: dailyinput.NumCell(row, 5, "hm_start", excel.ParseNumber, &errs),
		HmEnd:   dailyinput.NumCell(row, 6, "hm_end", excel.ParseNumber, &errs),
		HmTotal: dailyinput.NumCell(row, 7, "hm_total", excel.ParseNumber, &errs),
	}
	t := domain.TimeSheet{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyTimeSheet(pd, &t, in)...), timeSheetColumns)
	return t.OperationalDate + "|" + t.ShiftCode + "|" + t.EquipmentCode, t, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.TimeSheet](recs), by, dryRun)
}
