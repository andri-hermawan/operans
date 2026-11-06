// Package service (fitur importexcel) = Upload Excel: template, pratinjau (dry run) dan simpan untuk
// 4 jenis data harian. Fitur ini hanya MENGATUR alurnya (baca file → periksa tiap baris → simpan sekaligus);
// cara mengurai & menyimpan baris tiap jenis ada di service fitur pemiliknya (operator, activation,
// breakdown, weighbridge — file service/import.go), disambungkan di cmd/api/main.go.
package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/importexcel/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
)

// RowImporter = satu jenis impor: template + cara mengurai baris + cara menyimpan.
// Dipenuhi service fitur operator, activation, breakdown, dan weighbridge.
type RowImporter interface {
	// ImportColumns = judul kolom template + satu baris contoh.
	ImportColumns() (headers []string, example []any)
	// ImportKey = keterangan kunci "data sama ditimpa" untuk pesan peringatan.
	ImportKey() string
	// ParseImportRow mengubah satu baris menjadi record + kunci "data sama ditimpa".
	ParseImportRow(pd *dailyinput.ProjectData, row []string) (key string, rec any, errs []dailyinput.RowIssue)
	// SaveImportRows menyimpan semua record dalam satu transaksi (dryRun=true → dibatalkan).
	SaveImportRows(ctx context.Context, recs []any, by uuid.UUID, dryRun bool) (created, updated int, err error)
}

const maxIssues = 500

type Service struct {
	daily     dailyinput.Loader
	importers map[string]RowImporter // jenis impor (bagian URL /imports/:type) → fitur pemiliknya
}

func New(daily dailyinput.Loader, importers map[string]RowImporter) *Service {
	return &Service{daily: daily, importers: importers}
}

// Types = jenis impor yang tersedia.
func (s *Service) Types() []string {
	out := make([]string, 0, len(s.importers))
	for k := range s.importers {
		out = append(out, k)
	}
	return out
}

// Template mengembalikan file .xlsx template (sama dengan sistem lama).
func (s *Service) Template(typ string) ([]byte, error) {
	spec, ok := s.importers[typ]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	headers, example := spec.ImportColumns()
	return excel.Template(headers, example)
}

// Import membaca file, memeriksa setiap baris, lalu (bila dryRun=false dan tidak ada kesalahan)
// menyimpan semua baris dalam satu transaksi.
func (s *Service) Import(ctx context.Context, a access.Actor, typ string, projectID *uuid.UUID,
	filename string, file io.Reader, dryRun bool) (*domain.ImportResult, error) {

	spec, ok := s.importers[typ]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	pd, err := s.daily.Load(ctx, a, projectID, true)
	if err != nil {
		return nil, err
	}
	rows, err := excel.ReadTable(filename, file)
	if err != nil {
		return nil, &apperror.ValidationError{Msg: err.Error()}
	}

	res := &domain.ImportResult{Type: typ, DryRun: dryRun, Errors: []dailyinput.RowIssue{}, Warnings: []dailyinput.RowIssue{}}
	keyRow := map[string]int{}   // kunci → nomor baris yang dipakai
	keyIndex := map[string]int{} // kunci → posisi di recs
	var recs []any
	errCount := 0

	for i, row := range rows {
		rowNo := i + 2 // baris 1 = judul
		if isBlankRow(row) {
			continue
		}
		res.TotalRows++
		key, rec, errs := spec.ParseImportRow(pd, row)
		if len(errs) > 0 {
			res.InvalidRows++
			for _, e := range errs {
				errCount++
				if len(res.Errors) < maxIssues {
					e.Row = rowNo
					res.Errors = append(res.Errors, e)
				}
			}
			continue
		}
		if prev, dup := keyRow[key]; dup { // kunci sama di file → baris terakhir menang (seperti sistem lama)
			res.DuplicateRows++
			recs[keyIndex[key]] = rec
			if len(res.Warnings) < maxIssues {
				res.Warnings = append(res.Warnings, dailyinput.RowIssue{Row: rowNo,
					Message: fmt.Sprintf("menimpa baris %d (%s sama)", prev, spec.ImportKey())})
			}
			keyRow[key] = rowNo
			continue
		}
		keyRow[key], keyIndex[key] = rowNo, len(recs)
		recs = append(recs, rec)
	}
	res.ValidRows = res.TotalRows - res.InvalidRows
	if errCount > len(res.Errors) {
		res.Warnings = append(res.Warnings, dailyinput.RowIssue{Message: fmt.Sprintf("%d kesalahan lain tidak ditampilkan", errCount-len(res.Errors))})
	}
	if res.TotalRows == 0 {
		return nil, apperror.Invalid("file tidak berisi data")
	}
	if res.InvalidRows > 0 {
		if dryRun {
			return res, nil
		}
		return res, apperror.ErrImportHasErrors
	}

	// Pratinjau juga menjalankan penyimpanan di transaksi yang DIBATALKAN, sehingga hitungan
	// baru/diperbarui akurat dan aturan database sudah ikut diperiksa.
	res.ToCreate, res.ToUpdate, err = spec.SaveImportRows(ctx, recs, a.UserID, dryRun)
	if err != nil {
		return nil, err
	}
	res.Saved = !dryRun
	return res, nil
}

func isBlankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
