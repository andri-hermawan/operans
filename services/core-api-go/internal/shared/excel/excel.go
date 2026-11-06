// Package excel = membaca tabel dari file Excel/CSV, mengurai sel (tanggal, jam, angka),
// dan membuat file template. Tidak tahu soal database maupun aturan bisnis.
package excel

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	MaxFileBytes = 5 << 20 // 5 MB
	MaxDataRows  = 10_000  // di luar baris judul
)

// ReadTable membaca SHEET PERTAMA (.xlsx) atau seluruh file (.csv) menjadi baris-baris teks.
// Baris 1 = judul dan TIDAK dikembalikan (template dibaca berdasarkan urutan kolom, ADR 0007).
// Nilai sel diambil MENTAH: sel tanggal/jam Excel menjadi angka seri (mis. "46235", "0.2916…").
// Kembalian rows[i] = baris file nomor i+2 (nomor yang dilihat user di Excel).
func ReadTable(filename string, r io.Reader) ([][]string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileBytes {
		return nil, fmt.Errorf("file maksimal %d MB", MaxFileBytes>>20)
	}

	var rows [][]string
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".xlsx":
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			return nil, errors.New("file .xlsx tidak bisa dibaca (rusak atau bukan Excel)")
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("file tidak memiliki sheet")
		}
		if rows, err = f.GetRows(sheets[0], excelize.Options{RawCellValue: true}); err != nil {
			return nil, err
		}
	case ".csv":
		cr := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")))) // buang BOM Excel
		cr.FieldsPerRecord = -1
		if rows, err = cr.ReadAll(); err != nil {
			return nil, fmt.Errorf("CSV tidak valid: %w", err)
		}
	case ".xls":
		return nil, errors.New("format .xls lama tidak didukung; buka di Excel lalu simpan sebagai .xlsx")
	default:
		return nil, errors.New("format file harus .xlsx atau .csv")
	}

	if len(rows) <= 1 {
		return nil, errors.New("file tidak berisi data (baris 1 adalah judul kolom)")
	}
	rows = rows[1:]
	for len(rows) > 0 && blank(rows[len(rows)-1]) { // abaikan baris kosong di ujung
		rows = rows[:len(rows)-1]
	}
	if len(rows) > MaxDataRows {
		return nil, fmt.Errorf("maksimal %d baris data per file", MaxDataRows)
	}
	return rows, nil
}

// Cell mengambil kolom ke-i (0-based) yang sudah dipangkas; "" bila tidak ada.
func Cell(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

func blank(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// excelEpoch = hari ke-0 sistem tanggal Excel (sudah memperhitungkan bug tahun kabisat 1900).
var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// ParseDate menerima: sel tanggal Excel (angka seri), "11-08-2026", "11/08/2026", "11.08.2026",
// "2026-08-11", "2026/08/11". Hasil: "YYYY-MM-DD".
func ParseDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("tanggal wajib diisi")
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f < 1 || f > 2958465 { // 1900-01-01 .. 9999-12-31
			return "", fmt.Errorf("tanggal %q tidak valid", s)
		}
		return excelEpoch.AddDate(0, 0, int(math.Floor(f))).Format("2006-01-02"), nil
	}
	norm := strings.NewReplacer("/", "-", ".", "-").Replace(s)
	for _, layout := range []string{"2-1-2006", "2006-1-2"} {
		if t, err := time.Parse(layout, norm); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("tanggal %q tidak dikenali (contoh: 11-08-2026)", s)
}

// ParseClock menerima "07:00", "7:00", "07:00:00", atau sel jam Excel (pecahan hari, mis. 0.2916…).
// ok=false bila kosong.
func ParseClock(s string) (hh, mm int, ok bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !strings.Contains(s, ":") {
		frac := f - math.Floor(f) // sel tanggal+jam: ambil bagian jamnya saja
		total := int(math.Round(frac * 24 * 60))
		if total == 24*60 {
			total = 0
		}
		return total / 60, total % 60, true, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) >= 2 {
		h, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 == nil && e2 == nil && h >= 0 && h < 24 && m >= 0 && m < 60 {
			return h, m, true, nil
		}
	}
	return 0, 0, false, fmt.Errorf("jam %q tidak valid (contoh: 07:00)", s)
}

// ParseNumber: kosong → nil. Koma dianggap pemisah ribuan seperti sistem lama ("46,200" = 46200).
func ParseNumber(s string) (*float64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, fmt.Errorf("angka %q tidak valid", s)
	}
	return &f, nil
}

// ParseDuration menerima "1:30" / "01:30:00" (jam:menit) atau sel jam Excel (pecahan hari,
// mis. 0.0625 = 01:30). Hasil dalam JAM, dibulatkan 2 desimal; nil bila kosong. Maks. 24 jam.
func ParseDuration(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	bad := fmt.Errorf("durasi %q tidak valid (tulis jam:menit, contoh: 01:30)", s)
	var minutes float64
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		h, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil || h < 0 || m < 0 || m >= 60 || len(parts) > 3 {
			return nil, bad
		}
		minutes = float64(h*60 + m)
	} else {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f < 0 {
			return nil, bad
		}
		minutes = math.Round(f * 24 * 60)
	}
	if minutes > 24*60 {
		return nil, bad
	}
	h := math.Round(minutes/60*100) / 100
	return &h, nil
}

// Template membuat file .xlsx (sheet "Template") berisi baris judul + satu baris contoh,
// sama dengan template sistem lama (ADR 0007).
func Template(headers []string, example []any) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Template"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, v)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
