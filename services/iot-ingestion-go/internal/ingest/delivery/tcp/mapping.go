package tcp

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/pkg/teltonika"
)

// ID elemen IO Teltonika yang dipetakan ke kolom SIMHA, sesuai tabel mapping FMC125 & FMC150
// dari Teltonika Configurator (2026-10-22). IO yang tidak dikirim suatu model otomatis bernilai nil.
//
// Posisi, kecepatan, ketinggian, arah & jumlah satelit diambil dari elemen GPS bawaan record
// (raw.latitude, raw.speed, ...). IO 387 (ISO6709 Coordinates) dan IO 24 (Speed) di FMC150
// berisi informasi yang sama, jadi cukup disimpan di metadata.io.
const (
	ioAnalogInput1    = 9   // mV                                   FMC125, FMC150
	ioTotalOdometer   = 16  // meter (dari GPS)                     FMC125, FMC150
	ioAxisX           = 17  // mG, bertanda                         FMC125, FMC150
	ioAxisY           = 18  // mG, bertanda                         FMC125, FMC150
	ioAxisZ           = 19  // mG, bertanda                         FMC125, FMC150
	ioGSMSignal       = 21  // 0..5                                 FMC125, FMC150
	ioExternalVolt    = 66  // mV                                   FMC125, FMC150
	ioBatteryVolt     = 67  // mV                                   FMC125, FMC150
	ioBatteryAmp      = 68  // mA                                   FMC125, FMC150
	ioGNSSStatus      = 69  // 0 mati, 1 fix, 2 tanpa fix, 3 tidur  FMC125, FMC150
	ioTotalMileageCAN = 87  // meter (dari ECU lewat CAN)           FMC150
	ioPDOP            = 181 // ×10                                  FMC125, FMC150
	ioHDOP            = 182 // ×10                                  FMC125, FMC150
	ioSleepMode       = 200 //                                      FMC125, FMC150
	ioLLSFuelLevel    = 201 // nilai mentah sensor fuel             FMC125 ⚠ KONFIRMASI di Configurator
	ioLLSTemperature  = 202 // °C, bertanda (1 byte)                FMC125
	ioIgnition        = 239 // 1 = mesin hidup                      FMC125, FMC150
	ioMovement        = 240 // 1 = bergerak                         FMC125, FMC150
	ioGSMOperator     = 241 // kode MCC-MNC                         FMC125, FMC150
	// Fuel level dari CAN (FMC150): ID belum diketahui → sementara hanya ada di metadata.io.
)

// IOSettings = ID AVL yang BISA DIGANTI tanpa mengubah kode (diisi dari .env di cmd/iot/main.go).
type IOSettings struct {
	// FuelPercent = ID AVL persen level tangki dari ECU (unit bertipe percentage). 0 = tidak dibaca.
	// Bawaan 89 ("Fuel Level %", dokumentasi Teltonika); ubah lewat FUEL_PERCENT_IO_ID di .env bila di alat lain.
	FuelPercent uint16
}

// DefaultIO = pengaturan bawaan (dipakai tes dan config).
var DefaultIO = IOSettings{FuelPercent: 89}

// percentOf membaca IO persen (apa adanya, 0–100; 255 = galat ECU akan ditolak oleh domain.InvalidFuelPercent).
func percentOf(rec teltonika.Record, id uint16) *float64 {
	if id == 0 {
		return nil
	}
	return scaled(rec, id, 1)
}

// ToEvent mengubah satu AVL record menjadi domain.Event standar SIMHA.
// SEMUA IO mentah tetap disimpan di metadata.io, jadi tidak ada data yang hilang
// walaupun pemetaan di atas belum lengkap.
func ToEvent(imei string, codec byte, rec teltonika.Record, io IOSettings) domain.Event {
	lat, lon := rec.Latitude, rec.Longitude
	alt := int32(rec.AltitudeM)
	heading := int16(rec.AngleDeg)
	speed := float64(rec.SpeedKph)
	sats := int16(rec.Satellites)

	e := domain.Event{
		// Waktu ms + ID pemicu: sama persis bila alat mengirim ulang record yang sama → dianggap duplikat.
		EventID:   fmt.Sprintf("%d-%d", rec.Timestamp.UnixMilli(), rec.EventIOID),
		IMEI:      imei,
		EventTime: rec.Timestamp,
		Position: domain.Position{
			Latitude: &lat, Longitude: &lon, AltitudeM: &alt, HeadingDeg: &heading,
			SpeedKph: &speed, SatelliteCount: &sats,
			HDOP:       scaled(rec, ioHDOP, 0.1),
			PDOP:       scaled(rec, ioPDOP, 0.1),
			GNSSStatus: int16Of(rec, ioGNSSStatus),
		},
		Engine: domain.Engine{
			IsEngineOn: boolOf(rec, ioIgnition),
			IsMoving:   boolOf(rec, ioMovement),
			MileageM:   scaled(rec, ioTotalMileageCAN, 1),
			OdometerM:  scaled(rec, ioTotalOdometer, 1),
		},
		Fuel: domain.Fuel{
			FuelLevelRaw:     scaled(rec, ioLLSFuelLevel, 1), // MENTAH, bukan liter (liter setelah kalibrasi, Step 15)
			FuelTemperatureC: signed8(rec, ioLLSTemperature),
		},
		FuelSensorPercent: percentOf(rec, io.FuelPercent),
		Power: domain.Power{
			ExternalVoltageV: scaled(rec, ioExternalVolt, 0.001),
			BatteryVoltageV:  scaled(rec, ioBatteryVolt, 0.001),
			BatteryCurrentA:  scaled(rec, ioBatteryAmp, 0.001),
			GSMSignalLevel:   int16Of(rec, ioGSMSignal),
			GSMOperatorCode:  int32Of(rec, ioGSMOperator),
			SleepMode:        int16Of(rec, ioSleepMode),
		},
		Sensors: domain.Sensors{
			AccelXmg:      signed16(rec, ioAxisX),
			AccelYmg:      signed16(rec, ioAxisY),
			AccelZmg:      signed16(rec, ioAxisZ),
			AnalogInput1V: scaled(rec, ioAnalogInput1, 0.001),
		},
		Metadata: map[string]any{
			"source":      "teltonika",
			"codec":       fmt.Sprintf("0x%02X", codec),
			"priority":    rec.Priority,
			"event_io_id": rec.EventIOID,
			"io":          rawIO(rec),
		},
	}
	return e
}

// rawIO menyalin semua IO apa adanya: {"239": 1, "66": 27450, "387": "+-03.47+103.84/", "385": "aabb"}.
// IO panjang variabel disimpan sebagai teks bila isinya teks biasa (mis. IO 387), selain itu sebagai hex.
func rawIO(rec teltonika.Record) map[string]any {
	out := make(map[string]any, len(rec.IO))
	for _, e := range rec.IO {
		key := strconv.Itoa(int(e.ID))
		switch {
		case e.Size != 0:
			out[key] = e.Value
		case isPrintable(e.Bytes):
			out[key] = string(e.Bytes)
		default:
			out[key] = hex.EncodeToString(e.Bytes)
		}
	}
	return out
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return len(b) > 0
}

// Helper: kembalikan nil bila IO tidak dikirim alat (nil ≠ 0, aturan Step 7).

func scaled(rec teltonika.Record, id uint16, factor float64) *float64 {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	f := float64(v) * factor
	return &f
}

func signed8(rec teltonika.Record, id uint16) *float64 {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	f := float64(int8(uint8(v)))
	return &f
}

func signed16(rec teltonika.Record, id uint16) *float64 {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	f := float64(int16(uint16(v)))
	return &f
}

func boolOf(rec teltonika.Record, id uint16) *bool {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	b := v != 0
	return &b
}

func int16Of(rec teltonika.Record, id uint16) *int16 {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	x := int16(v)
	return &x
}

func int32Of(rec teltonika.Record, id uint16) *int32 {
	v, ok := rec.IOValue(id)
	if !ok {
		return nil
	}
	x := int32(v)
	return &x
}
