package tcp

import (
	"testing"
	"time"

	"simha/iot-ingestion/pkg/teltonika"
)

var ts = time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)

// Record contoh FMC125 (sensor fuel Escort lewat IO 201/202).
func recordFMC125() teltonika.Record {
	return teltonika.Record{
		Timestamp: ts, Longitude: 103.8491, Latitude: -3.47, AltitudeM: 85, AngleDeg: 270, Satellites: 12, SpeedKph: 32,
		EventIOID: ioIgnition,
		IO: []teltonika.IOElement{
			{ID: ioIgnition, Size: 1, Value: 1},
			{ID: ioExternalVolt, Size: 2, Value: 27450},  // 27,45 V
			{ID: ioHDOP, Size: 1, Value: 9},              // 0,9
			{ID: ioAxisX, Size: 2, Value: 0xFFF6},        // -10 mG
			{ID: ioLLSFuelLevel, Size: 2, Value: 2048},   // fuel mentah
			{ID: ioLLSTemperature, Size: 1, Value: 0xFB}, // -5 °C
			{ID: 270, Size: 2, Value: 1234},              // belum dipetakan
		},
	}
}

// Record contoh FMC150 (data ECU lewat CAN).
func recordFMC150() teltonika.Record {
	return teltonika.Record{
		Timestamp: ts, Longitude: 103.8491, Latitude: -3.47, Satellites: 14, SpeedKph: 40,
		IO: []teltonika.IOElement{
			{ID: ioIgnition, Size: 1, Value: 1},
			{ID: ioTotalMileageCAN, Size: 4, Value: 152340500},      // meter
			{ID: ioTotalOdometer, Size: 4, Value: 98765432},         // meter
			{ID: 24, Size: 2, Value: 40},                            // Speed (sama dengan raw.speed)
			{ID: 387, Bytes: []byte("+-03.4700+103.8491+085.000/")}, // ISO6709
		},
	}
}

func TestToEventFMC125(t *testing.T) {
	e := ToEvent("463719065679574", teltonika.Codec8E, recordFMC125(), DefaultIO)

	if e.EventID != "1792656000000-239" || !e.EventTime.Equal(ts) {
		t.Errorf("event_id=%s time=%s", e.EventID, e.EventTime)
	}
	if *e.Latitude != -3.47 || *e.Longitude != 103.8491 || *e.SpeedKph != 32 || *e.HeadingDeg != 270 {
		t.Error("posisi/kecepatan salah")
	}
	if e.IsEngineOn == nil || !*e.IsEngineOn {
		t.Error("ignition seharusnya true")
	}
	if *e.ExternalVoltageV != 27.45 || *e.HDOP != 0.9 || *e.AccelXmg != -10 {
		t.Errorf("konversi satuan salah: V=%v hdop=%v ax=%v", *e.ExternalVoltageV, *e.HDOP, *e.AccelXmg)
	}
	if e.FuelLevelRaw == nil || *e.FuelLevelRaw != 2048 || e.FuelTemperatureC == nil || *e.FuelTemperatureC != -5 {
		t.Errorf("fuel salah: level=%v suhu=%v", e.FuelLevelRaw, e.FuelTemperatureC)
	}
	if e.IsMoving != nil || e.OdometerM != nil || e.MileageM != nil {
		t.Error("IO yang tidak dikirim harus nil, bukan 0")
	}
	if e.Metadata["io"].(map[string]any)["270"] != uint64(1234) {
		t.Error("IO yang belum dipetakan harus tetap tersimpan di metadata.io")
	}
	if err := e.Validate(ts); err != nil {
		t.Fatalf("event hasil pemetaan seharusnya valid: %v", err)
	}
}

func TestToEventFMC150(t *testing.T) {
	e := ToEvent("463719065679575", teltonika.Codec8E, recordFMC150(), DefaultIO)

	if e.MileageM == nil || *e.MileageM != 152340500 || e.OdometerM == nil || *e.OdometerM != 98765432 {
		t.Errorf("mileage/odometer salah: %v / %v", e.MileageM, e.OdometerM)
	}
	if e.FuelLevelRaw != nil || e.FuelTemperatureC != nil {
		t.Error("FMC150 tidak mengirim IO 201/202 → fuel harus nil")
	}
	io := e.Metadata["io"].(map[string]any)
	if io["387"] != "+-03.4700+103.8491+085.000/" {
		t.Errorf("IO 387 seharusnya tersimpan sebagai teks, dapat %v", io["387"])
	}
	if err := e.Validate(ts); err != nil {
		t.Fatalf("event hasil pemetaan seharusnya valid: %v", err)
	}
}

func TestFuelPercentIO(t *testing.T) {
	rec := recordFMC150()
	rec.IO = append(rec.IO, teltonika.IOElement{ID: 89, Size: 1, Value: 63}) // AVL 89 = Fuel Level %
	if e := ToEvent("463719065679575", teltonika.Codec8E, rec, DefaultIO); e.FuelSensorPercent == nil || *e.FuelSensorPercent != 63 {
		t.Fatalf("persen dari IO 89: %v", e.FuelSensorPercent)
	}
	// ID bisa diganti lewat pengaturan (FUEL_PERCENT_IO_ID)
	if e := ToEvent("463719065679575", teltonika.Codec8E, rec, IOSettings{FuelPercent: 90}); e.FuelSensorPercent != nil {
		t.Error("IO 90 tidak ada di record, persen harus kosong")
	}
	// 0 = tidak dibaca
	if e := ToEvent("463719065679575", teltonika.Codec8E, rec, IOSettings{}); e.FuelSensorPercent != nil {
		t.Error("ID 0 berarti tidak dibaca")
	}
}
