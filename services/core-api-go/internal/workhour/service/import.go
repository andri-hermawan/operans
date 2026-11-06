package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/workhour/domain"
)

// ---------------------------------------------------------------------
// Impor Excel WH & Notification (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "CnUnit", "Category", "SubCategory", "Start", "Stop", "TotalHours", "Dedicated", "Remark"},
		[]any{"01/05/2025", "Shift 1", "DZ20034", "Standby", "P5M", "07:00", "07:15", 0.25, "", ""}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift, unit, kategori & jam mulai" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var workHourColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "equipment_code": "CnUnit",
	"category": "Category", "sub_category": "SubCategory", "time_start": "Start", "time_end": "Stop",
	"total_hours": "TotalHours", "dedicated": "Dedicated", "remark": "Remark"}

// ParseImportRow — WH & Notification: Date | Shift | CnUnit | Category | SubCategory | Start | Stop | TotalHours | Dedicated | Remark.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := WorkHourInput{
		OperationalDate: cell(0), Shift: cell(1), EquipmentCode: cell(2), Category: cell(3), SubCategory: cell(4),
		TimeStart: cell(5), TimeEnd: cell(6), Dedicated: cell(8), Remark: cell(9),
		TotalHours: dailyinput.NumCell(row, 7, "total_hours", excel.ParseNumber, &errs),
	}
	t := domain.WorkHourNotification{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyWorkHour(pd, &t, in)...), workHourColumns)
	return t.OperationalDate + "|" + t.ShiftCode + "|" + t.EquipmentCode + "|" + t.Category + "|" + t.StartedAt.Format("15:04"), t, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.WorkHourNotification](recs), by, dryRun)
}
