// Package domain (fitur report) = bahan laporan matriks Asset ID × tanggal (Ritase/Payload/Produksi/Breakdown/PA).
package domain

import "github.com/google/uuid"

// DailyCell = satu unit pada satu tanggal operasional (bahan laporan matriks Asset ID × tanggal).
type DailyCell struct {
	EquipmentCode string  `json:"equipment_code"`
	Date          string  `json:"date"`
	Count         int     `json:"count"`  // weighbridge: ritase (jumlah tiket); breakdown: jumlah catatan
	SumKg         float64 `json:"sum_kg"` // weighbridge: total net kg
	SumHours      float64 `json:"sum_hours"`
}

// ExecutiveRow = produksi (net kg) satu customer pada tanggal laporan, bulan berjalan (MTD), dan tahun berjalan (YTD).
type ExecutiveRow struct {
	Customer string  `json:"customer"`
	DailyKg  float64 `json:"daily_kg"`
	MtdKg    float64 `json:"mtd_kg"`
	YtdKg    float64 `json:"ytd_kg"`
}

// ExecutiveEmplacementRow = per partner (stok emplacement, ton): produksi = Total TLS Out (harian/MTD/YTD) dan
// inventory = Sisa Cargo pada tanggal laporan.
type ExecutiveEmplacementRow struct {
	Partner   string  `json:"partner"`
	DailyProd float64 `json:"daily_production"`
	MtdProd   float64 `json:"mtd_production"`
	YtdProd   float64 `json:"ytd_production"`
	Inventory float64 `json:"inventory"`
}

// ReportFilter = periode + shift/unit opsional dalam satu project.
type ReportFilter struct {
	ProjectID uuid.UUID
	From, To  string
	ShiftID   *uuid.UUID
	Codes     []string
}
