package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/stockemplacement/domain"
)

// ---------------------------------------------------------------------
// Impor Excel stok cargo (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// importTitles = judul kolom template (sama dengan sistem lama), urutan kolom Excel.
var importTitles = []string{"Partner", "Operational Date", "Stock Cargo Awal", "Stock Cargo Scrapper", "Stock Cargo Stockpile Crusher",
	"Info", "Shift 1", "Shift 2", "Cargo Hauling", "Total Cargo Awal", "TLS Out Trainset 1", "TLS Out Trainset 2",
	"TLS Out Trainset 3", "TLS Out Trainset 4", "TLS Out Trainset 5", "Total TLS Out", "Sisa Cargo"}

// ImportColumns = judul kolom template + satu baris contoh. Angka dalam ton.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return importTitles, []any{"CPM", "01/10/2026", 36877.59, 36877.59, 0, "OK", 0, 0, 0, 36877.59, 2874.16, 0, 0, 0, 0, 2874.16, 34003.43}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "partner & tanggal" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var stockColumns = func() map[string]string {
	m := map[string]string{"partner": "Partner", "operational_date": "Operational Date", "info": "Info"}
	for i, c := range domain.AmountColumns {
		m[c] = importTitles[amountCell(i)]
	}
	return m
}()

// amountCell = nomor kolom Excel untuk angka ke-i (Info ada di antara angka ke-3 dan ke-4).
func amountCell(i int) int {
	if i < 3 {
		return i + 2
	}
	return i + 3
}

// ParseImportRow — Stock Emplacement: Partner | Operational Date | 3 angka stok | Info | 11 angka (urutan domain.AmountColumns).
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	in := StockEmplacementInput{Partner: dailyinput.PtrStr(excel.Cell(row, 0)), OperationalDate: dailyinput.PtrStr(excel.Cell(row, 1)),
		Info: dailyinput.PtrStr(excel.Cell(row, 5))}
	for i, p := range []**float64{&in.StockInitial, &in.StockScrapper, &in.StockCrusher, &in.Shift1, &in.Shift2,
		&in.HaulingCargo, &in.TotalInitial, &in.TlsOut1, &in.TlsOut2, &in.TlsOut3, &in.TlsOut4, &in.TlsOut5,
		&in.TotalTlsOut, &in.RemainingCargo} {
		*p = dailyinput.NumCell(row, amountCell(i), domain.AmountColumns[i], excel.ParseNumber, &errs)
	}
	e := domain.StockEmplacement{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyStock(&e, in)...), stockColumns)
	return e.Partner + "|" + e.OperationalDate, e, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.StockEmplacement](recs), by, dryRun)
}
