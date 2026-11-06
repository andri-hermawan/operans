// Package domain = aturan murni pipeline penerimaan data (tanpa database/HTTP): bentuk Event dan
// validasinya, status operasi, status muatan, zona, fuel, alert, geofence, shift, dan kondisi terakhir
// (LiveState). Satu topik satu file. Struct di sini juga dipakai fitur lain sebagai bentuk data
// (mis. livestate, notification, telemetry).
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Position = kelompok 2 tabel: posisi GNSS.
// Latitude + Longitude disimpan ke kolom `location` (PostGIS) oleh repository.
type Position struct {
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	AltitudeM      *int32   `json:"altitude_m"`
	HeadingDeg     *int16   `json:"heading_deg"`
	SpeedKph       *float64 `json:"speed_kph"`
	SatelliteCount *int16   `json:"satellite_count"`
	HDOP           *float64 `json:"hdop"`
	PDOP           *float64 `json:"pdop"`
	GNSSStatus     *int16   `json:"gnss_status"`
}

// Engine = kelompok 3: mesin & jarak.
type Engine struct {
	IsEngineOn *bool    `json:"is_engine_on"`
	IsMoving   *bool    `json:"is_moving"`
	MileageM   *float64 `json:"mileage_m"`
	OdometerM  *float64 `json:"odometer_m"`
	// Muatan dari alat (ton), lama: equipment_logs.vessel. Bukan status muatan (itu dihitung server).
	VesselLoadTon *float64 `json:"vessel_load_ton"`
}

// Fuel = kelompok 4. FuelLevelRaw adalah angka MENTAH sensor, bukan liter.
type Fuel struct {
	FuelLevelRaw     *float64 `json:"fuel_level_raw"`
	FuelVolumeLiter  *float64 `json:"fuel_volume_liter"`
	FuelPercent      *float64 `json:"fuel_percent"`
	FuelDeltaLiter   *float64 `json:"fuel_delta_liter"`
	FuelTemperatureC *float64 `json:"fuel_temperature_c"`
}

// Power = kelompok 5: kelistrikan & modem.
type Power struct {
	ExternalVoltageV *float64 `json:"external_voltage_v"`
	BatteryVoltageV  *float64 `json:"battery_voltage_v"`
	BatteryCurrentA  *float64 `json:"battery_current_a"`
	GSMSignalLevel   *int16   `json:"gsm_signal_level"`
	GSMOperatorCode  *int32   `json:"gsm_operator_code"`
	SleepMode        *int16   `json:"sleep_mode"`
}

// Sensors = kelompok 6: sensor tambahan.
type Sensors struct {
	AccelXmg      *float64 `json:"accel_x_mg"`
	AccelYmg      *float64 `json:"accel_y_mg"`
	AccelZmg      *float64 `json:"accel_z_mg"`
	AnalogInput1V *float64 `json:"analog_input_1_v"`
}

// Event = satu kiriman data dari alat, dalam bentuk standar SIMHA.
// Kelompok-kelompok di-embed, sehingga JSON-nya datar dan nama key = nama kolom tabel.
type Event struct {
	EventID   string    `json:"event_id"`
	IMEI      string    `json:"imei"`
	EventTime time.Time `json:"event_time"`
	Position
	Engine
	Fuel
	// Persen level tangki dari ECU (AVL ID 89 / FUEL_PERCENT_IO_ID), khusus unit bertipe percentage. Masukan sensor:
	// tidak disimpan sendiri; oleh ApplyFuel diubah menjadi fuel_level_raw (setara), liter, persen, dan selisih.
	FuelSensorPercent *float64 `json:"fuel_sensor_percent,omitempty"`
	Power
	Sensors
	Metadata map[string]any `json:"metadata"`
	Derived
}

// Derived = nilai yang DIHITUNG server, bukan dikirim alat.
// Tag json:"-" membuatnya tidak bisa diisi dari kiriman (dan kiriman yang mencoba akan ditolak sebagai field asing).
type Derived struct {
	EquipmentStatus *string `json:"-"` // running | idle | stop | offline

	// Dari data induk core (Fase C1, ADR 0009); kosong bila data induk belum ada
	OperationalDate      *string `json:"-"` // YYYY-MM-DD = tanggal mulai shift
	ShiftCode            *string `json:"-"`
	OperatorName         *string `json:"-"`
	IsBreakdown          *bool   `json:"-"`
	BreakdownDescription *string `json:"-"`

	// Zona peta & status muatan (Fase C2); GeofenceFID nil = posisi/zona tidak diketahui
	ZoneID           *uuid.UUID `json:"-"`
	IsInsideGeofence *bool      `json:"-"`
	GeofenceFID      *int32     `json:"-"` // ORIG_FID zona; 0 = di luar semua zona
	LocationCategory *string    `json:"-"`
	SegmentName      *string    `json:"-"`
	VesselStatus     *string    `json:"-"` // loaded | empty | unknown

	// Acuan status muatan berikutnya (zona terakhir di dalam peta); disimpan di live state, bukan telemetry
	LastInsideFID          *int32  `json:"-"`
	LastInsideVesselStatus *string `json:"-"`

	// Keadaan pendeteksi kejadian SETELAH data ini (Fase C3/C4); nil = tidak berubah (data terlambat)
	Tracker    *Tracker `json:"-"`
	AlertCount *int     `json:"-"` // jumlah alert unit pada tanggal operasional + shift ini (C4)
}

const (
	maxFutureSkew = 5 * time.Minute
	maxEventIDLen = 100 // sama dengan varchar(100) di tabel
)

var (
	ErrInvalid   = errors.New("event tidak valid")
	imeiPattern  = regexp.MustCompile(`^[0-9]{15}$`)
	minEventTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

func invalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalid, msg)
}

type number interface {
	~int16 | ~int32 | ~float64
}

// checkRange: nil (tidak dikirim) selalu lolos; nilai yang ada harus di dalam min..max.
func checkRange[T number](field string, v *T, min, max T) error {
	if v != nil && (*v < min || *v > max) {
		return invalid(fmt.Sprintf("%s=%v di luar rentang %v..%v", field, *v, min, max))
	}
	return nil
}

// Normalize merapikan data sebelum divalidasi. Tidak menolak apa pun.
func (e *Event) Normalize() {
	e.EventTime = e.EventTime.UTC()
	// Teltonika mengirim 0,0 saat belum mendapat fix GPS: itu "tidak ada posisi", bukan posisi nyata.
	if e.Latitude != nil && e.Longitude != nil && *e.Latitude == 0 && *e.Longitude == 0 {
		e.Latitude, e.Longitude = nil, nil
	}
	// Mileage (dipakai Activity Summary) disamakan dengan odometer alat: tidak semua unit punya mileage dari ECU (CAN),
	// sedangkan odometer GPS selalu ada. Nilai mentah mileage CAN tetap tersimpan di metadata.io.
	if e.OdometerM != nil {
		m := *e.OdometerM
		e.MileageM = &m
	}
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
}

// Validate mengembalikan nil bila event layak disimpan.
// Setiap error membungkus ErrInvalid, jadi bisa dicek dengan errors.Is(err, ErrInvalid).
func (e *Event) Validate(now time.Time) error {
	// 1) Aturan dasar: kalau salah satu gagal, langsung tolak.
	switch {
	case e.EventID == "":
		return invalid("event_id wajib")
	case len(e.EventID) > maxEventIDLen:
		return invalid("event_id maksimal 100 karakter")
	case !imeiPattern.MatchString(e.IMEI):
		return invalid("imei harus tepat 15 digit angka")
	case e.EventTime.IsZero():
		return invalid("event_time wajib (format RFC3339, mis. 2026-10-22T08:15:00Z)")
	case e.EventTime.Before(minEventTime):
		return invalid("event_time sebelum 2020: jam alat belum sinkron")
	case e.EventTime.After(now.Add(maxFutureSkew)):
		return invalid("event_time di masa depan")
	case (e.Latitude == nil) != (e.Longitude == nil):
		return invalid("latitude dan longitude harus dikirim berpasangan")
	}

	// 2) Rentang nilai wajar: semua pelanggaran dilaporkan sekaligus.
	return errors.Join(
		checkRange("latitude", e.Latitude, -90, 90),
		checkRange("longitude", e.Longitude, -180, 180),
		checkRange("heading_deg", e.HeadingDeg, 0, 360),
		checkRange("speed_kph", e.SpeedKph, 0, 200),
		checkRange("satellite_count", e.SatelliteCount, 0, 64),
		checkRange("hdop", e.HDOP, 0, 100),
		checkRange("pdop", e.PDOP, 0, 100),
		checkRange("mileage_m", e.MileageM, 0, 9_999_999_999.99),
		checkRange("odometer_m", e.OdometerM, 0, 9_999_999_999_999.99),
		checkRange("vessel_load_ton", e.VesselLoadTon, 0, 500),
		checkRange("fuel_percent", e.FuelPercent, 0, 100),
		checkRange("external_voltage_v", e.ExternalVoltageV, 0, 60),
		checkRange("gsm_signal_level", e.GSMSignalLevel, 0, 5),
	)
}
