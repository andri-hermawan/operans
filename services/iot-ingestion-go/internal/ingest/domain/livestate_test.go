package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOperationalStatus(t *testing.T) {
	sig, noSig := int16(4), int16(0)
	op := int32(51010)
	on, off := true, false
	moving, still := 25.0, 0.0

	tests := []struct {
		name string
		e    Event
		want string
	}{
		{"sinyal GSM 0 → offline", Event{Power: Power{GSMSignalLevel: &noSig, GSMOperatorCode: &op}, Engine: Engine{IsEngineOn: &on}}, StatusOffline},
		{"operator GSM kosong → offline", Event{Power: Power{GSMSignalLevel: &sig}, Engine: Engine{IsEngineOn: &on}}, StatusOffline},
		{"mesin hidup & bergerak → running", Event{Power: Power{GSMSignalLevel: &sig, GSMOperatorCode: &op}, Engine: Engine{IsEngineOn: &on}, Position: Position{SpeedKph: &moving}}, StatusRunning},
		{"mesin hidup & diam → idle", Event{Power: Power{GSMSignalLevel: &sig, GSMOperatorCode: &op}, Engine: Engine{IsEngineOn: &on}, Position: Position{SpeedKph: &still}}, StatusIdle},
		{"mesin mati → stop", Event{Power: Power{GSMSignalLevel: &sig, GSMOperatorCode: &op}, Engine: Engine{IsEngineOn: &off}}, StatusStop},
		{"mesin tidak dikirim → stop", Event{Power: Power{GSMSignalLevel: &sig, GSMOperatorCode: &op}}, StatusStop},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OperationalStatus(&tt.e); got != tt.want {
				t.Fatalf("status = %s, seharusnya %s", got, tt.want)
			}
		})
	}
}

func TestNextLiveState(t *testing.T) {
	owner := Owner{EquipmentID: uuid.New(), DeviceID: uuid.New()}
	t0 := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC)
	lat, lon, fuel, speed := -3.47, 103.85, 2500.0, 20.0
	running := StatusRunning

	first := Event{EventTime: t0, Position: Position{Latitude: &lat, Longitude: &lon, SpeedKph: &speed},
		Fuel: Fuel{FuelLevelRaw: &fuel}, Derived: Derived{EquipmentStatus: &running}}

	s1, changed := NextLiveState(nil, owner, &first, 10, t0, t0)
	if !changed || s1.TelemetryEventID != 10 || *s1.FuelLevelRaw != 2500 || *s1.EquipmentStatus != StatusRunning {
		t.Fatalf("event pertama: %+v", s1)
	}

	t.Run("event baru tanpa posisi & fuel: posisi dan fuel terakhir tetap", func(t *testing.T) {
		e := Event{EventTime: t0.Add(30 * time.Second)}
		s2, changed := NextLiveState(&s1, owner, &e, 11, t0, t0)
		if !changed || s2.TelemetryEventID != 11 || *s2.Latitude != lat || *s2.FuelLevelRaw != 2500 {
			t.Fatalf("hasil: %+v", s2)
		}
		if s2.SpeedKph != nil || s2.EquipmentStatus != nil {
			t.Fatal("speed & status selalu mengikuti event terbaru")
		}
	})

	t.Run("data terlambat tidak menimpa kondisi terbaru", func(t *testing.T) {
		old := Event{EventTime: t0.Add(-time.Hour)}
		s3, changed := NextLiveState(&s1, owner, &old, 9, t0, t0)
		if changed || s3.TelemetryEventID != 10 {
			t.Fatalf("changed=%v id=%d", changed, s3.TelemetryEventID)
		}
	})
}
