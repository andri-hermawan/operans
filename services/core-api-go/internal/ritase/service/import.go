package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"simha/core-api/internal/ritase/domain"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Impor Excel Ritase (dipanggil fitur importexcel; kolom berdasarkan URUTAN template, ADR 0007)
// ---------------------------------------------------------------------

// ImportColumns = judul kolom template (sama dengan tabel lama) + satu baris contoh.
func (s *Service) ImportColumns() (headers []string, example []any) {
	return []string{"Date", "Shift", "Time", "Loader", "LoaderModel", "NikOperatorLoader", "OperatorLoader", "UserLoader",
			"Hauler", "HaulerModel", "NikOperatorHauler", "OperatorHauler", "UserHauler",
			"Blok", "Location", "Dumping", "Material", "SubMaterial", "Seam", "Phase",
			"Distance", "Ritase", "Prod", "ProdAdj", "DistxVol", "ProductivityFactor"},
		[]any{"01/05/2025", "Shift 1", "07.00-08.00", "EX001", "", "", "", "TBBE", "DT001", "", "", "", "TBBE",
			"", "", "", "", "", "", "", 0, 0, 0, 0, 0, 0}
}

// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
func (s *Service) ImportKey() string { return "tanggal, shift, jam, loader, hauler, lokasi & material" }

// Nama isian API → judul kolom template, untuk pesan kesalahan.
var ritaseColumns = map[string]string{"operational_date": "Date", "shift": "Shift", "time_slot": "Time",
	"loader_code": "Loader", "loader_model": "LoaderModel", "loader_operator_nik": "NikOperatorLoader",
	"loader_operator_name": "OperatorLoader", "loader_partner": "UserLoader",
	"hauler_code": "Hauler", "hauler_model": "HaulerModel", "hauler_operator_nik": "NikOperatorHauler",
	"hauler_operator_name": "OperatorHauler", "hauler_partner": "UserHauler",
	"block": "Blok", "location": "Location", "dumping": "Dumping", "material": "Material", "sub_material": "SubMaterial",
	"seam": "Seam", "phase": "Phase", "distance": "Distance", "ritase": "Ritase", "production": "Prod",
	"production_adjusted": "ProdAdj", "distance_volume": "DistxVol", "productivity_factor": "ProductivityFactor"}

// ParseImportRow — Ritase: kolom sesuai ImportColumns (26 kolom).
func (s *Service) ParseImportRow(pd *dailyinput.ProjectData, row []string) (string, any, []dailyinput.RowIssue) {
	var errs []dailyinput.RowIssue
	cell := func(i int) *string { return dailyinput.PtrStr(excel.Cell(row, i)) }
	num := func(i int, col string) *float64 { return dailyinput.NumCell(row, i, col, excel.ParseNumber, &errs) }
	in := RitaseInput{
		OperationalDate: cell(0), Shift: cell(1), TimeSlot: cell(2),
		LoaderCode: cell(3), LoaderModel: cell(4), LoaderOperatorNik: cell(5), LoaderOperatorName: cell(6), LoaderPartner: cell(7),
		HaulerCode: cell(8), HaulerModel: cell(9), HaulerOperatorNik: cell(10), HaulerOperatorName: cell(11), HaulerPartner: cell(12),
		Block: cell(13), Location: cell(14), Dumping: cell(15), Material: cell(16), SubMaterial: cell(17), Seam: cell(18), Phase: cell(19),
		Distance: num(20, "distance"), Ritase: num(21, "ritase"), Production: num(22, "production"),
		ProductionAdjusted: num(23, "production_adjusted"), DistanceVolume: num(24, "distance_volume"),
		ProductivityFactor: num(25, "productivity_factor"),
	}
	t := domain.Ritase{ProjectID: pd.Project.ID}
	errs = dailyinput.RenameColumns(append(errs, applyRitase(pd, &t, in)...), ritaseColumns)
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s", t.OperationalDate, t.ShiftCode, t.TimeSlot, t.LoaderCode, t.HaulerCode,
		str(t.Block), str(t.Location), str(t.Dumping), str(t.Material), str(t.SubMaterial), str(t.Seam), str(t.Phase))
	return key, t, errs
}

// SaveImportRows menyimpan semua baris dalam SATU transaksi (dryRun=true → dibatalkan, hanya menghitung).
func (s *Service) SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (int, int, error) {
	return s.repo.BulkUpsert(ctx, dailyinput.Records[domain.Ritase](recs), by, dryRun)
}
