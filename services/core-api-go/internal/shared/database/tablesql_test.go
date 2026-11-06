package database

import "testing"

func TestTableSQLNomorParameterSejalanDenganKolom(t *testing.T) {
	tb := Table{Name: "core.contoh", Key: "uq_contoh_key", Cols: []string{"operational_date", "nama", "jumlah"},
		Casts: map[string]string{"operational_date": "::text::date"}}

	wantUpsert := "INSERT INTO core.contoh (project_id, operational_date, nama, jumlah, created_by, updated_by)\n" +
		"VALUES ($1, $2::text::date, $3, $4, $5, $5)\n" +
		"ON CONFLICT ON CONSTRAINT uq_contoh_key\n" +
		"DO UPDATE SET operational_date = EXCLUDED.operational_date, nama = EXCLUDED.nama, jumlah = EXCLUDED.jumlah, updated_by = EXCLUDED.updated_by\n" +
		"RETURNING id, (xmax = 0)"
	if got := tb.UpsertSQL(); got != wantUpsert {
		t.Errorf("UpsertSQL salah:\n%s\n--- ingin:\n%s", got, wantUpsert)
	}

	wantUpdate := "UPDATE core.contoh\n   SET operational_date = $2::text::date, nama = $3, jumlah = $4, updated_by = $5\n WHERE id = $1"
	if got := tb.UpdateSQL(); got != wantUpdate {
		t.Errorf("UpdateSQL salah:\n%s\n--- ingin:\n%s", got, wantUpdate)
	}
}
