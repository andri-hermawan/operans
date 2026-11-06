// simulator-dt03 meniru simulator lama FMS/BACKEND/fms-backend/simulator-api/simulator-api-DT03.js untuk SIMHA:
// skenario yang sama (150 data: posisi, fuel, speed, mesin; jam data tiap 30 detik mulai hari ini 07:05:00 WIB),
// dikirim SATU PER SATU tiap 3 detik ke IoT SIMHA, dengan rumus odometer & nilai acak yang sama dengan simulator lama.
// Tidak membaca fms_lite sama sekali — skenario ada di skenario-dt03.json (disalin dari file simulator lama).
//
//	go run ./tools/simulator-dt03 -key <API key alat>
//	go run ./tools/simulator-dt03 -key <API key alat> -jeda 0          # tanpa menunggu 3 detik
//	go run ./tools/simulator-dt03 -key <API key alat> -mulai "2026-10-27 07:05:00"   # tanggal lain (bawaan: hari ini)
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

//go:embed skenario-dt03.json
var skenarioJSON []byte

// satu baris skenario (sama dengan isi array "scenario" simulator lama setelah scenarioUpdates diterapkan)
type langkah struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	FuelLevel    float64 `json:"fuel_level"`
	Speed        float64 `json:"speed"`
	EngineStatus bool    `json:"engine_status"`
	GSMSignal    int     `json:"gsm_signal"`
	GSMOperator  int     `json:"gsm_operator"`
}

func main() {
	iot := flag.String("iot", "http://localhost:5550", "alamat IoT SIMHA")
	key := flag.String("key", os.Getenv("SIMULATOR_API_KEY"), "API key alat di iot.devices (X-API-Key)")
	hariIni := time.Now().In(wibZone()).Format("2006-01-02")
	mulai := flag.String("mulai", hariIni+" 07:05:00", "jam data pertama (WIB). Bawaan: hari ini 07:05:00 (tanggal ikut hari berjalan, seperti simulator DT07)")
	jarak := flag.Duration("jarak", 30*time.Second, "selisih jam antar data (SCENARIO_INTERVAL_SECONDS lama)")
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
	start, err := time.ParseInLocation("2006-01-02 15:04:05", mulai, wibZone())
	if err != nil {
		return fmt.Errorf("-mulai: %w", err)
	}
	var skenario []langkah
	if err := json.Unmarshal(skenarioJSON, &skenario); err != nil {
		return err
	}

	fmt.Println("===========================================")
	fmt.Println("SIMHA SIMULATOR DT03 (meniru simulator lama)")
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
		fmt.Printf("#%03d %s | Engine=%s | Speed=%2.0f | Fuel=%v | ", i+1, t.Format("15:04:05"), mesin, s.Speed, s.FuelLevel)
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

// buatData = buildBody simulator lama, dengan nama field SIMHA.
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
		ftemp = bulat(acak(29, 34), 1)
	}
	return map[string]any{
		// event_id dari jam data: kirim ulang skenario yang sama = duplikat, bukan data ganda
		"event_id":           "dt03-" + t.UTC().Format("20060102T150405Z"),
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
		"pdop":               bulat(acak(0.8, 1.5), 2),
		"hdop":               bulat(acak(0.8, 1.5), 2),
		"gnss_status":        pilih(1, 0),
		"fuel_level_raw":     s.FuelLevel,
		"fuel_temperature_c": ftemp,
		"sleep_mode":         pilih(0, 1),
		"analog_input_1_v":   0,
		"vessel_load_ton":    0,
		"metadata":           map[string]any{"source": "simulator-dt03", "movement_runtime": (i + 1) * 3},
	}
}

func acak(min, max float64) float64 { return rand.Float64()*(max-min) + min }
func acakInt(min, max int) int      { return min + rand.Intn(max-min+1) }
func bulat(v float64, d int) float64 {
	p := math.Pow(10, float64(d))
	return math.Round(v*p) / p
}
