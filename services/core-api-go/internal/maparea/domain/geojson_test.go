package domain

import (
	"strings"
	"testing"
)

const square = `{"type":"Polygon","coordinates":[[[103.85,-3.63],[103.86,-3.63],[103.86,-3.62],[103.85,-3.62],[103.85,-3.63]]]}`

func feature(props, geom string) string {
	return `{"type":"Feature","properties":` + props + `,"geometry":` + geom + `}`
}

func collection(features ...string) string {
	return `{"type":"FeatureCollection","features":[` + strings.Join(features, ",") + `]}`
}

func TestParseFormatLama(t *testing.T) {
	// atribut persis seperti peta CHR di sistem lama
	file := collection(feature(`{"CODE":"25","CODE_2":25,"Segment":"Km 23+000 - 24+000","Category":"Trase Utama",
		"HEX_Fill":"#d3ffbe","HEX_Line":"#000000","ORIG_FID":25,"BUFF_DIST":15}`, square))
	raw, feats, issues, err := Parse(strings.NewReader(file))
	if err != nil || len(issues) != 0 || len(feats) != 1 {
		t.Fatalf("err=%v issues=%v feats=%d", err, issues, len(feats))
	}
	f := feats[0]
	if f.Category != "Trase Utama" || f.Name != "Km 23+000 - 24+000" || *f.SourceFID != 25 ||
		*f.FillColor != "#D3FFBE" || *f.LineColor != "#000000" || f.Properties["BUFF_DIST"] != float64(15) {
		t.Fatalf("feature: %+v", f)
	}
	if strings.Contains(string(raw), "\n") {
		t.Fatal("file asli seharusnya disimpan ringkas")
	}
}

func TestParseKesalahanPerFeature(t *testing.T) {
	open := `{"type":"Polygon","coordinates":[[[103.85,-3.63],[103.86,-3.63],[103.86,-3.62],[103.85,-3.62]]]}`
	utm := `{"type":"Polygon","coordinates":[[[450000,9600000],[450100,9600000],[450100,9600100],[450000,9600000]]]}`
	line := `{"type":"LineString","coordinates":[[103.85,-3.63],[103.86,-3.63]]}`
	file := collection(
		feature(`{"Category":"Emplacement","Segment":"Office"}`, square),         // 1 benar
		feature(`{"Segment":"Tanpa kategori"}`, square),                          // 2 Category kosong
		feature(`{"Category":"A","Segment":"Terbuka"}`, open),                    // 3 cincin tidak tertutup
		feature(`{"Category":"A","Segment":"UTM"}`, utm),                         // 4 koordinat bukan derajat
		feature(`{"Category":"A","Segment":"Garis"}`, line),                      // 5 bukan polygon
		feature(`{"Category":"A","Segment":"Warna","HEX_Fill":"hijau"}`, square), // 6 warna salah
		feature(`{"Category":"A","Segment":"Prioritas","Priority":5}`, square),   // 7 benar, priority 5
	)
	_, feats, issues, err := Parse(strings.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{2: "Category", 3: "geometry", 4: "geometry", 5: "geometry", 6: "HEX_Fill"}
	if len(issues) != len(want) || len(feats) != 2 || feats[1].Priority != 5 {
		t.Fatalf("issues=%+v feats=%d", issues, len(feats))
	}
	for _, i := range issues {
		if want[i.Feature] != i.Field {
			t.Errorf("kesalahan tak terduga: %+v", i)
		}
	}
}

func TestParseFileDitolak(t *testing.T) {
	for name, file := range map[string]string{
		"bukan json":       `ini bukan json`,
		"bukan collection": feature(`{}`, square),
		"kosong":           collection(),
		"crs utm":          `{"type":"FeatureCollection","crs":{"type":"name","properties":{"name":"urn:ogc:def:crs:EPSG::32748"}},"features":[]}`,
	} {
		if _, _, _, err := Parse(strings.NewReader(file)); err == nil {
			t.Errorf("%s: seharusnya ditolak", name)
		}
	}
	crs84 := `{"type":"FeatureCollection","crs":{"type":"name","properties":{"name":"urn:ogc:def:crs:OGC:1.3:CRS84"}},"features":[` +
		feature(`{"Category":"A","Segment":"B"}`, square) + `]}`
	if _, feats, _, err := Parse(strings.NewReader(crs84)); err != nil || len(feats) != 1 {
		t.Fatalf("CRS84 (ekspor QGIS) seharusnya diterima: %v", err)
	}
}
