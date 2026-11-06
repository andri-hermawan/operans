package excel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseDate(t *testing.T) {
	ok := map[string]string{
		"11-08-2026": "2026-08-11", "1/8/2026": "2026-08-01", "11.08.2026": "2026-08-11",
		"2026-08-11": "2026-08-11", "2026/8/1": "2026-08-01",
		"46245": "2026-08-11", // sel tanggal Excel
	}
	for in, want := range ok {
		if got, err := ParseDate(in); err != nil || got != want {
			t.Errorf("%q → %q, %v (seharusnya %q)", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "32-01-2026", "besok", "2026-13-01"} {
		if _, err := ParseDate(bad); err == nil {
			t.Errorf("%q seharusnya ditolak", bad)
		}
	}
}

func TestParseClock(t *testing.T) {
	cases := map[string][2]int{"07:00": {7, 0}, "7:05": {7, 5}, "23:59:00": {23, 59}, "0.5": {12, 0}, "46245.75": {18, 0}}
	for in, want := range cases {
		h, m, ok, err := ParseClock(in)
		if err != nil || !ok || h != want[0] || m != want[1] {
			t.Errorf("%q → %d:%d ok=%v err=%v", in, h, m, ok, err)
		}
	}
	if _, _, ok, err := ParseClock(""); ok || err != nil {
		t.Error("kosong = tidak ada jam, bukan error")
	}
	if _, _, _, err := ParseClock("25:00"); err == nil {
		t.Error("25:00 seharusnya ditolak")
	}
}

func TestParseNumber(t *testing.T) {
	if n, _ := ParseNumber("46,200"); *n != 46200 {
		t.Errorf("46,200 → %v", *n)
	}
	if n, _ := ParseNumber(""); n != nil {
		t.Error("kosong → nil")
	}
	if _, err := ParseNumber("abc"); err == nil {
		t.Error("abc seharusnya ditolak")
	}
}

func TestTemplateDanBaca(t *testing.T) {
	b, err := Template([]string{"Tanggal", "Shift", "Asset ID"}, []any{"11-08-2026", "Shift 1", "EQ-001"})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := excelize.OpenReader(bytes.NewReader(b))
	if f.GetSheetList()[0] != "Template" {
		t.Fatal("nama sheet harus Template")
	}
	rows, err := ReadTable("template.xlsx", bytes.NewReader(b))
	if err != nil || len(rows) != 1 || rows[0][1] != "Shift 1" {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
}

func TestCSVdanFormatDitolak(t *testing.T) {
	csv := "\xEF\xBB\xBFTanggal,Shift\n11-08-2026,\"Shift 1\"\n,\n"
	rows, err := ReadTable("data.csv", strings.NewReader(csv))
	if err != nil || len(rows) != 1 || rows[0][0] != "11-08-2026" {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if _, err := ReadTable("lama.xls", strings.NewReader("x")); err == nil {
		t.Fatal(".xls lama seharusnya ditolak dengan pesan jelas")
	}
}
