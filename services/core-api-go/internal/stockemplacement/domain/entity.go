// Package domain (fitur stockemplacement) = stok cargo harian per project (satuan ton), satu baris per partner per tanggal operasional.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AmountColumns = nama kolom angka (ton) di tabel core.stock_emplacements, urutan sama dengan StockEmplacement.Amounts()
// dan dengan urutan kolom template Excel.
var AmountColumns = []string{
	"stock_cargo_initial", "stock_cargo_scrapper", "stock_cargo_crusher", "shift_1", "shift_2",
	"hauling_cargo", "total_cargo_initial",
	"tls_out_trainset_1", "tls_out_trainset_2", "tls_out_trainset_3", "tls_out_trainset_4", "tls_out_trainset_5",
	"total_tls_out", "remaining_cargo",
}

// StockEmplacement = stok cargo satu hari (tabel core.stock_emplacements; lama: StockEmplacement). Semua angka ton, boleh kosong.
// Kolom total disimpan apa adanya dari input.
type StockEmplacement struct {
	ID              int64     `json:"id"`
	ProjectID       uuid.UUID `json:"project_id"`
	Partner         string    `json:"partner"`
	OperationalDate string    `json:"operational_date"`
	Info            *string   `json:"info"`
	StockInitial    *float64  `json:"stock_cargo_initial"`
	StockScrapper   *float64  `json:"stock_cargo_scrapper"`
	StockCrusher    *float64  `json:"stock_cargo_crusher"`
	Shift1          *float64  `json:"shift_1"`
	Shift2          *float64  `json:"shift_2"`
	HaulingCargo    *float64  `json:"hauling_cargo"`
	TotalInitial    *float64  `json:"total_cargo_initial"`
	TlsOut1         *float64  `json:"tls_out_trainset_1"`
	TlsOut2         *float64  `json:"tls_out_trainset_2"`
	TlsOut3         *float64  `json:"tls_out_trainset_3"`
	TlsOut4         *float64  `json:"tls_out_trainset_4"`
	TlsOut5         *float64  `json:"tls_out_trainset_5"`
	TotalTlsOut     *float64  `json:"total_tls_out"`
	RemainingCargo  *float64  `json:"remaining_cargo"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Amounts = penunjuk ke setiap kolom angka, urutan = AmountColumns.
func (e *StockEmplacement) Amounts() []**float64 {
	return []**float64{&e.StockInitial, &e.StockScrapper, &e.StockCrusher, &e.Shift1, &e.Shift2,
		&e.HaulingCargo, &e.TotalInitial, &e.TlsOut1, &e.TlsOut2, &e.TlsOut3, &e.TlsOut4, &e.TlsOut5,
		&e.TotalTlsOut, &e.RemainingCargo}
}
