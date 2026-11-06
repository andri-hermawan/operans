package database

import (
	"fmt"
	"strings"
)

// Table menyusun SQL tambah/ubah sebuah tabel input dari DAFTAR KOLOM, sehingga nomor parameter ($1, $2, …) tidak
// pernah diketik tangan dan selalu sejalan dengan urutan kolom. Dipakai fitur input transaksi (time sheet, ritase, …).
//
// Urutan argumen:
//   - UpsertSQL : project_id, Cols…, by           (by dipakai untuk created_by dan updated_by)
//   - UpdateSQL : id, Cols…, by                   (project_id tidak diubah)
type Table struct {
	Name  string            // mis. core.time_sheets
	Key   string            // nama constraint unik "data sama ditimpa"
	Cols  []string          // kolom isian setelah project_id, tanpa id dan kolom audit
	Casts map[string]string // kolom → akhiran cast parameter, mis. operational_date → "::text::date"
}

func (t Table) param(n int, col string) string { return fmt.Sprintf("$%d%s", n, t.Casts[col]) }

// UpsertSQL: INSERT … ON CONFLICT (kunci) DO UPDATE … RETURNING id, (baris baru?) — format yang dipakai BulkUpsert.
func (t Table) UpsertSQL() string {
	vals := []string{"$1"}
	sets := make([]string, len(t.Cols))
	for i, c := range t.Cols {
		vals = append(vals, t.param(i+2, c))
		sets[i] = c + " = EXCLUDED." + c
	}
	by := len(t.Cols) + 2
	return fmt.Sprintf("INSERT INTO %s (project_id, %s, created_by, updated_by)\nVALUES (%s, $%d, $%d)\nON CONFLICT ON CONSTRAINT %s\nDO UPDATE SET %s, updated_by = EXCLUDED.updated_by\nRETURNING id, (xmax = 0)",
		t.Name, strings.Join(t.Cols, ", "), strings.Join(vals, ", "), by, by, t.Key, strings.Join(sets, ", "))
}

// UpdateSQL: UPDATE satu baris berdasarkan id ($1).
func (t Table) UpdateSQL() string {
	sets := make([]string, len(t.Cols))
	for i, c := range t.Cols {
		sets[i] = fmt.Sprintf("%s = %s", c, t.param(i+2, c))
	}
	return fmt.Sprintf("UPDATE %s\n   SET %s, updated_by = $%d\n WHERE id = $1", t.Name, strings.Join(sets, ", "), len(t.Cols)+2)
}
