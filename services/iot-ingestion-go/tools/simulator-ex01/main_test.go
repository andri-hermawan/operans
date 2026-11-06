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
	if len(sk) != 20 {
		t.Fatalf("skenario DT04 lama berisi 20 data, dapat %d", len(sk))
	}
	for i, s := range sk {
		if s.FuelLevel < 0 || s.FuelLevel > 4095 {
			t.Fatalf("data #%d: level mentah %v di luar 0–4095", i+1, s.FuelLevel)
		}
	}
	if sk[0].FuelLevel != 1970 || sk[9].FuelLevel != 2937.467 { // pengisian BBM di data ke-10 dan ke-11
		t.Errorf("level BBM: awal=%v setelah isi=%v", sk[0].FuelLevel, sk[9].FuelLevel)
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
	if e.FuelLevelRaw == nil || *e.FuelLevelRaw != sk[3].FuelLevel {
		t.Errorf("level mentah tidak terkirim: %v", e.FuelLevelRaw)
	}
}
