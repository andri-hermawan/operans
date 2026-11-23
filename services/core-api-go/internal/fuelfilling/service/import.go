package service

import (
	"context"
	"strconv"

	"github.com/google/uuid"

	"simha/core-api/internal/fuelfilling/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel Fuel Filling (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template (sama dengan tabel lama, tanpa id/ProjectCode/Status/User/CreateDate/UpdateDate) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "Model", "CnUnit", "HmPengisianSebelumnya", "HmPengisian", "TotalHours", "FuelUsageLtr",
			"FuelConsLtrHrs", "WasteRemoval", "CoalMining", "Hauling", "Rental", "Dewatering", "FinalSupport"},
		[]any{"01/08/2025", "Shift 1", "D8T", "DZ30053", 5085.2, 5095, 9.8, 255, 26.02, 255, 0, 0, 0, 0, 0}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift, unit & HM pengisian" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var fuelFillingColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "equipment_model": "Model", "equipment_code": "CnUnit",
	"hm_previous": "HmPengisianSebelumnya", "hm_filling": "HmPengisian", "total_hours": "TotalHours", "fuel_usage_liter": "FuelUsageLtr",
	"fuel_consumption_lph": "FuelConsLtrHrs", "waste_removal_liter": "WasteRemoval", "coal_mining_liter": "CoalMining",
	"hauling_liter": "Hauling", "rental_liter": "Rental", "dewatering_liter": "Dewatering", "final_support_liter": "FinalSupport"}

// ParseImportRow — Fuel Filling: Date | Shift | Model | CnUnit | HmPengisianSebelumnya | HmPengisian | TotalHours | FuelUsageLtr |
// FuelConsLtrHrs | WasteRemoval | CoalMining | Hauling | Rental | Dewatering | FinalSupport.
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	num := func(i int, col string) *float64 { return dailyinput.NumCell(row, i, col, excel.ParseNumber, &errs) }
	in := FuelFillingInput{
		OperationalDate: cell(0), Shift: cell(1), EquipmentModel: cell(2), EquipmentCode: cell(3),
		HmPrevious: num(4, "hm_previous"), HmFilling: num(5, "hm_filling"), TotalHours: num(6, "total_hours"),
		FuelUsage: num(7, "fuel_usage_liter"), FuelConsumption: num(8, "fuel_consumption_lph"),
		WasteRemoval: num(9, "waste_removal_liter"), CoalMining: num(10, "coal_mining_liter"), Hauling: num(11, "hauling_liter"),
		Rental: num(12, "rental_liter"), Dewatering: num(13, "dewatering_liter"), FinalSupport: num(14, "final_support_liter"),
	}
	f := domain.FuelFilling{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyFuelFilling(pd, &f, in)...), fuelFillingColumns)
	hm := "-"
	if f.HmFilling != nil {
		hm = strconv.FormatFloat(*f.HmFilling, 'f', 2, 64)
	}
	return f.OperationalDate + "|" + f.ShiftCode + "|" + f.EquipmentCode + "|" + hm, f, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.FuelFilling](recs), by, dryRun)
}
