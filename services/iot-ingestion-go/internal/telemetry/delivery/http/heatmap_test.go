package http

import "testing"

func TestParseHeatGrid(t *testing.T) {
	q, msg := parseHeatGrid("", "", "")
	if msg != "" || q.CellMeters != defaultCell || q.HasBBox || q.Limit != defaultHeatCell {
		t.Fatalf("bawaan salah: %+v %q", q, msg)
	}
	q, msg = parseHeatGrid("40", "103.8,-3.6,103.9,-3.4", "500")
	if msg != "" || q.CellMeters != 40 || !q.HasBBox || q.MinLat != -3.6 || q.MaxLon != 103.9 || q.Limit != 500 {
		t.Fatalf("parsing salah: %+v %q", q, msg)
	}
	for _, c := range [][3]string{{"1", "", ""}, {"abc", "", ""}, {"20", "1,2,3", ""}, {"20", "103.9,-3.6,103.8,-3.4", ""}, {"20", "a,b,c,d", ""}} {
		if _, msg := parseHeatGrid(c[0], c[1], c[2]); msg == "" {
			t.Errorf("seharusnya ditolak: %v", c)
		}
	}
	if q, _ := parseHeatGrid("", "", "999999"); q.Limit != defaultHeatCell {
		t.Errorf("limit di atas batas harus diabaikan: %d", q.Limit)
	}
}
