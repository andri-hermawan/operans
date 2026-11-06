// simulator-ex01 meniru simulator lama FMS/BACKEND/fms-backend/simulator-api/simulator-api-DT04.js untuk SIMHA,
// untuk equipment 48258b4f-2bef-43c6-a78b-7ca2eaf6ee07 (EX00001, Excavator; sensor BBM LLS → level mentah 0–4095):
// skenario yang sama (20 data tiap 30 detik, selisih 9,5 menit dari data pertama ke terakhir, termasuk pengisian BBM di data ke-10 dan ke-11),
// dikirim SATU PER SATU tiap 3 detik ke IoT SIMHA, dengan rumus odometer & nilai acak yang sama dengan simulator lama.
//
// Beda dengan simulator lama: kirim ke IoT SIMHA (POST /api/v1/ingest/events, header X-API-Key), bukan ke FMS lama.
// Alat yang mengirim ditentukan oleh API key-nya (iot.devices.equipment_id), jadi alat dengan API key yang dipakai di sini
// harus sudah didaftarkan dan dihubungkan ke equipment 48258b4f-... Liter dan persen dihitung SERVER dari kalibrasi unit.
//
// Tidak membaca fms_lite sama sekali — skenario ada di skenario-ex01.json (disalin dari file simulator lama).
//
//	go run ./tools/simulator-ex01 -key <API key alat equipment 48258b4f-...>
//	go run ./tools/simulator-ex01 -key <API key> -jeda 0                      # tanpa menunggu 3 detik
//	go run ./tools/simulator-ex01 -key <API key> -mulai "2026-10-29 18:55:00"   # jam tertentu (harus sudah lewat)
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"
)

//go:embed skenario-ex01.json
var skenarioJSON []byte

const equipmentID = "48258b4f-2bef-43c6-a78b-7ca2eaf6ee07"

// satu baris skenario (isi array "scenario" simulator DT04 lama)
type langkah struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	FuelLevel    float64 `json:"fuel_level"` // nilai mentah LLS 0–4095
	Speed        float64 `json:"speed"`
	EngineStatus bool    `json:"engine_status"`
	GSMSignal    int     `json:"gsm_signal"`
	GSMOperator  int     `json:"gsm_operator"`
}

func main() {
	iot := flag.String("iot", "http://localhost:5550", "alamat IoT SIMHA")
	key := flag.String("key", os.Getenv("SIMULATOR_API_KEY"), "API key alat equipment "+equipmentID+" di iot.devices (X-API-Key)")
	mulai := flag.String("mulai", "", "jam data pertama (WIB), mis. \"2026-11-05 18:55:00\". Bawaan: kosong = mulai tepat sekian menit lalu supaya data terakhir jatuh di jam sekarang (server menolak data di masa depan)")
	jarak := flag.Duration("jarak", 30*time.Second, "selisih jam antar data (sama dengan skenario lama)")
	jeda := flag.Duration("jeda", 3*time.Second, "jeda pengiriman antar data (SEND_INTERVAL lama)")
	flag.Parse()
	if err := run(*iot, *key, *mulai, *jarak, *jeda); err != nil {
		fmt.Fprintln(os.Stderr, "GAGAL:", err)
		os.Exit(1)
	}
}

func wibZone() *time.Location {
	wib, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return wib
}

func run(iot, key, mulai string, jarak, jeda time.Duration) error {
	if key == "" {
		return fmt.Errorf("-key (API key alat) wajib diisi")
	}
	var skenario []langkah
	if err := json.Unmarshal(skenarioJSON, &skenario); err != nil {
		return err
	}
	// Skenario lama bertanggal "hari ini" pukul 18:55, tetapi server menolak data di masa depan. Tanpa -mulai,
	// data pertama dimundurkan agar data terakhir jatuh di jam sekarang (tanggalnya tetap hari ini).
	start := time.Now().In(wibZone()).Truncate(time.Second).Add(-time.Duration(len(skenario)-1) * jarak)
	if mulai != "" {
		var err error
		if start, err = time.ParseInLocation("2006-01-02 15:04:05", mulai, wibZone()); err != nil {
			return fmt.Errorf("-mulai: %w", err)
		}
	}

	fmt.Println("===========================================")
	fmt.Println("SIMHA SIMULATOR EX01 (EX00001, sensor LLS)")
	fmt.Println("Equipment:", equipmentID)
	fmt.Println("Skenario :", len(skenario), "data, mulai", start.Format("2006-01-02 15:04:05"), "WIB")
	fmt.Println("Jeda     :", jeda)
	fmt.Println("===========================================")

	client := &http.Client{Timeout: 15 * time.Second}
	odometer, ok, dup := 0.0, 0, 0
	for i, s := range skenario {
		if i > 0 {
			time.Sleep(jeda)
			odometer += s.Speed * (3 / 3.6) // rumus simulator lama (apa adanya)
		}
		t := start.Add(time.Duration(i) * jarak)
		body, _ := json.Marshal(buatData(i, s, t, odometer))
		req, _ := http.NewRequest(http.MethodPost, iot+"/api/v1/ingest/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", key)
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()

		mesin := "OFF"
		if s.EngineStatus {
			mesin = "ON "
		}
		fmt.Printf("#%02d %s | Engine=%s | Speed=%2.0f | Fuel=%v | ", i+1, t.Format("15:04:05"), mesin, s.Speed, s.FuelLevel)
		switch res.StatusCode {
		case http.StatusCreated:
			ok++
			fmt.Println("TERSIMPAN")
		case http.StatusOK:
			dup++
			fmt.Println("DUPLIKAT (sudah pernah dikirim)")
		default:
			fmt.Println("DITOLAK")
			return fmt.Errorf("data #%d ditolak: HTTP %d %s", i+1, res.StatusCode, b)
		}
	}
	fmt.Printf("RINGKASAN dikirim=%d tersimpan=%d duplikat=%d odometer=%.0f m\n", len(skenario), ok, dup, odometer)
	return nil
}

// buatData = buildBody simulator lama, dengan nama field SIMHA. Level BBM dikirim sebagai nilai mentah LLS.
func buatData(i int, s langkah, t time.Time, odometer float64) map[string]any {
	on := s.EngineStatus
	pilih := func(a, b float64) float64 {
		if on {
			return a
		}
		return b
	}
	heading := 0
	if s.Speed != 0 {
		heading = acakInt(170, 190)
	}
	ftemp := 0.0
	if on {
		ftemp = bulatDesimal(acak(29, 34), 1)
	}
	return map[string]any{
		// event_id dari jam data: kirim ulang skenario yang sama = duplikat, bukan data ganda
		"event_id":           "ex01-" + t.UTC().Format("20060102T150405Z"),
		"event_time":         t.UTC().Format(time.RFC3339),
		"latitude":           s.Latitude,
		"longitude":          s.Longitude,
		"altitude_m":         0,
		"heading_deg":        heading,
		"satellite_count":    acakInt(10, 16),
		"speed_kph":          s.Speed,
		"is_moving":          s.Speed > 0,
		"accel_x_mg":         acakInt(-2, 2),
		"accel_y_mg":         acakInt(-2, 2),
		"accel_z_mg":         acakInt(-2, 2),
		"odometer_m":         math.Round(odometer),
		"mileage_m":          math.Round(odometer),
		"is_engine_on":       on,
		"external_voltage_v": pilih(24.5, 0),
		"battery_voltage_v":  pilih(3.9, 0),
		"battery_current_a":  0,
		"gsm_signal_level":   s.GSMSignal,
		"gsm_operator_code":  s.GSMOperator,
		"pdop":               bulatDesimal(acak(0.8, 1.5), 2),
		"hdop":               bulatDesimal(acak(0.8, 1.5), 2),
		"gnss_status":        pilih(1, 0),
		// sensor LLS: nilai mentah 0–4095 + suhu BBM (0 saat mesin mati), seperti simulator lama
		"fuel_level_raw":     s.FuelLevel,
		"fuel_temperature_c": ftemp,
		"sleep_mode":         pilih(0, 1),
		"analog_input_1_v":   0,
		"vessel_load_ton":    0,
		"metadata":           map[string]any{"source": "simulator-ex01", "movement_runtime": (i + 1) * 3},
	}
}

func acak(min, max float64) float64 { return rand.Float64()*(max-min) + min }
func acakInt(min, max int) int      { return min + rand.Intn(max-min+1) }
func bulatDesimal(v float64, d int) float64 {
	p := math.Pow(10, float64(d))
	return math.Round(v*p) / p
}
