package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/weighbridge/domain"
)

// ---------------------------------------------------------------------
// Impor Excel weighbridge (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template + satu baris contoh (sama dengan template sistem lama).
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "Ticket No", "Asset ID", "Product", "Gross", "Tare", "Net", "Recipient",
			"Customer", "Transporter", "Gross Time", "Tare Time", "Gross Operator", "Tare Operator", "Description", "Location"},
		[]any{"11-08-2026", "Shift 1", "A2026081200198", "10419", "BATUBARA", 46200, 14680, 31520, "PT. RMKO",
			"PT. DBU", "PT. RMKO", "13:09", "13:20", "Yulius", "Tegar", "DBU", ""}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal & nomor tiket" }

var weighbridgeColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "ticket_no": "Ticket No",
	"equipment_code": "Asset ID", "product": "Product", "gross_kg": "Gross", "tare_kg": "Tare", "net_kg": "Net",
	"recipient": "Recipient", "customer": "Customer", "transporter": "Transporter", "gross_time": "Gross Time",
	"tare_time": "Tare Time", "gross_operator": "Gross Operator", "tare_operator": "Tare Operator",
	"description": "Description", "location": "Location"}

// ParseImportRow — Weighbridge: Date | Shift | Ticket No | Asset ID | Product | Gross | Tare | Net | Recipient |
// Customer | Transporter | Gross Time | Tare Time | Gross Operator | Tare Operator | Description | Location
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	in := WeighbridgeInput{
		OperationalDate: cell(0), Shift: cell(1), TicketNo: cell(2), EquipmentCode: cell(3), Product: cell(4),
		GrossKg:   dailyinput.NumCell(row, 5, "gross_kg", excel.ParseNumber, &errs),
		TareKg:    dailyinput.NumCell(row, 6, "tare_kg", excel.ParseNumber, &errs),
		NetKg:     dailyinput.NumCell(row, 7, "net_kg", excel.ParseNumber, &errs),
		Recipient: cell(8), Customer: cell(9), Transporter: cell(10), GrossTime: cell(11), TareTime: cell(12),
		GrossOperator: cell(13), TareOperator: cell(14), Description: cell(15), Location: cell(16),
	}
	w := domain.WeighbridgeTicket{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyWeighbridge(pd, &w, in)...), weighbridgeColumns)
	return w.OperationalDate + "|" + w.TicketNo, w, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.WeighbridgeTicket](recs), by, dryRun)
}
