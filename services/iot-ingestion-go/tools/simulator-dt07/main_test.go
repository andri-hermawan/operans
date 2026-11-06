package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"simha/iot-ingestion/internal/ingest/domain"
)

func TestSkenario(t *testing.T) {
	var sk []langkah
	if err := json.Unmarshal(skenarioJSON, &sk); err != nil {
		t.Fatal(err)
	}
	if len(sk) != 252 {
		t.Fatalf("skenario DT02 lama berisi 252 data, dapat %d", len(sk))
	}
	for i, s := range sk {
		if s.FuelPercent <= 0 || s.FuelPercent > 100 {
			t.Fatalf("data #%d: persen %v di luar 0–100", i+1, s.FuelPercent)
		}
	}
	if sk[0].FuelPercent != 48.1 || sk[10].FuelPercent != 95.8 { // 1970/4095 dan 3922,467/4095
		t.Errorf("konversi LLS→persen: awal=%v setelah isi BBM=%v", sk[0].FuelPercent, sk[10].FuelPercent)
	}
}

// Isi kiriman harus bisa dibaca server persis seperti POST /api/v1/ingest/events (field asing ditolak) dan lolos validasi.
func TestKiriman(t *testing.T) {
	var sk []langkah
	_ = json.Unmarshal(skenarioJSON, &sk)
	raw, _ := json.Marshal(buatData(3, sk[3], time.Now().Add(-time.Minute), 1200))

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var e domain.Event
	if err := dec.Decode(&e); err != nil {
		t.Fatal(err)
	}
	e.IMEI = "863719065713472"
	e.Normalize()
	if err := e.Validate(time.Now()); err != nil {
		t.Fatal(err)
	}
	if e.FuelSensorPercent == nil || *e.FuelSensorPercent != sk[3].FuelPercent {
		t.Errorf("persen tidak terkirim: %v", e.FuelSensorPercent)
	}
	if e.FuelLevelRaw != nil || e.FuelTemperatureC != nil {
		t.Error("sensor ECU: tidak boleh ada fuel_level_raw maupun suhu")
	}
}
