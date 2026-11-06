package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/breakdown/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel breakdown (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template + satu baris contoh (sama dengan template sistem lama).
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "Asset ID", "Class", "Status", "Category", "Time Start", "Time End",
			"Duration", "Repair Status", "Description", "Location"},
		[]any{"11-08-2026", "Shift 1", "10303", "DUMP TRUCK", "Breakdown", "Unsch Maintenance", "07:00", "", "",
			"On Progress", "BATTERY BERMASALAH", "STA 24"}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift, unit, kategori & jam mulai" }

var breakdownColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "equipment_code": "Asset ID",
	"repair_status": "Repair Status", "status": "Status", "category": "Category", "time_start": "Time Start",
	"time_end": "Time End", "duration_hours": "Duration", "description": "Description", "location": "Location"}

// ParseImportRow — Breakdown: Date | Shift | Asset ID | Class (diabaikan) | Status | Category | Time Start |
// Time End | Duration | Repair Status | Description | Location
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := BreakdownInput{
		OperationalDate: cell(0), Shift: cell(1), EquipmentCode: cell(2), Status: cell(4), Category: cell(5),
		TimeStart: cell(6), TimeEnd: cell(7), RepairStatus: cell(9), Description: cell(10), Location: cell(11),
		DurationHours: dailyinput.NumCell(row, 8, "duration_hours", excel.ParseDuration, &errs),
	}
	b := domain.EquipmentBreakdown{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyBreakdown(pd, &b, in)...), breakdownColumns)
	start := ""
	if b.StartedAt != nil {
		start = b.StartedAt.UTC().Format(time.RFC3339)
	}
	return strings.Join([]string{b.OperationalDate, b.ShiftID.String(), b.EquipmentCode, b.CategoryCode, start}, "|"), b, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.EquipmentBreakdown](recs), by, dryRun)
}
