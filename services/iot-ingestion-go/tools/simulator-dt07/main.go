// simulator-dt07 meniru simulator lama FMS/BACKEND/fms-backend/simulator-api/simulator-api-DT02.js untuk SIMHA,
// khusus alat ber-IMEI 863719065713472 (DT00070, kalibrasi tipe PERCENTAGE / sensor ECU):
// skenario yang sama (252 data tiap 30 detik: 07:00:00 s/d 09:05:30, termasuk pengisian BBM di data ke-10 dan ke-11),
// dikirim SATU PER SATU tiap 3 detik ke IoT SIMHA, dengan rumus odometer & nilai acak yang sama dengan simulator lama.
//
// BEDA dengan simulator lama: level BBM pada skenario sudah diubah dari nilai mentah LLS (0–4095) menjadi PERSEN (0–100)
// — nilai ini dikirim sebagai `fuel_sensor_percent` (sama dengan AVL ID 89 "Fuel Level %" dari ECU), tanpa
// fuel_level_raw dan tanpa suhu BBM (ECU umumnya tidak mengirim suhu). Liter, persen terkalibrasi, selisih, dan raw
// setara dihitung SERVER dari kalibrasi unit (lihat ADR 0008).
//
// Tidak membaca fms_lite sama sekali — skenario ada di skenario-dt07.json (disalin dari file simulator lama).
//
//	go run ./tools/simulator-dt07 -key <API key alat 863719065713472>
//	go run ./tools/simulator-dt07 -key <API key> -jeda 0                      # tanpa menunggu 3 detik
//	go run ./tools/simulator-dt07 -key <API key> -mulai "2026-10-29 07:00:00"
//	go run ./tools/simulator-dt07 -key <API key> -bulat                       # persen dibulatkan ke bilangan bulat, seperti ECU asli
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

//go:embed skenario-dt07.json
var skenarioJSON []byte

// satu baris skenario (isi array "scenario" simulator DT02 lama; fuel_level diubah menjadi fuel_percent 0–100)
type langkah struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	FuelPercent  float64 `json:"fuel_percent"`
	Speed        float64 `json:"speed"`
	EngineStatus bool    `json:"engine_status"`
	GSMSignal    int     `json:"gsm_signal"`
	GSMOperator  int     `json:"gsm_operator"`
}

func main() {
	iot := flag.String("iot", "http://localhost:3347", "alamat IoT SIMHA")
	key := flag.String("key", os.Getenv("SIMULATOR_API_KEY"), "API key alat 863719065713472 di iot.devices (X-API-Key)")
	hariIni := time.Now().In(wibZone()).Format("2006-01-02")
	mulai := flag.String("mulai", hariIni+" 07:00:00", "jam data pertama (WIB). Bawaan: hari ini 07:00:00, sama dengan simulator DT02 lama (TODAY)")
	jarak := flag.Duration("jarak", 30*time.Second, "selisih jam antar data (sama dengan skenario lama)")
	jeda := flag.Duration("jeda", 3*time.Second, "jeda pengiriman antar data (SEND_INTERVAL lama)")
	bulat := flag.Bool("bulat", false, "bulatkan persen ke bilangan bulat, seperti ECU asli (AVL ID 89 berukuran 1 byte)")
	flag.Parse()
	if err := run(*iot, *key, *mulai, *jarak, *jeda, *bulat); err != nil {
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

func run(iot, key, mulai string, jarak, jeda time.Duration, bulat bool) error {
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
	fmt.Println("SIMHA SIMULATOR DT07 (DT00070, sensor persen)")
	fmt.Println("IMEI     : 863719065713472")
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
		if bulat {
			s.FuelPercent = math.Round(s.FuelPercent)
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
		fmt.Printf("#%03d %s | Engine=%s | Speed=%2.0f | Fuel=%.1f%% | ", i+1, t.Format("15:04:05"), mesin, s.Speed, s.FuelPercent)
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

// buatData = buildBody simulator lama, dengan nama field SIMHA. Level BBM dikirim sebagai persen ECU.
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
	return map[string]any{
		// event_id dari jam data: kirim ulang skenario yang sama = duplikat, bukan data ganda
		"event_id":           "dt07-" + t.UTC().Format("20060102T150405Z"),
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
		// sensor ECU: hanya persen (0–100). Tidak ada fuel_level_raw LLS dan tidak ada fuel_temperature_c.
		"fuel_sensor_percent": s.FuelPercent,
		"sleep_mode":          pilih(0, 1),
		"analog_input_1_v":    0,
		"vessel_load_ton":     0,
		"metadata":            map[string]any{"source": "simulator-dt07", "movement_runtime": (i + 1) * 3},
	}
}

func acak(min, max float64) float64 { return rand.Float64()*(max-min) + min }
func acakInt(min, max int) int      { return min + rand.Intn(max-min+1) }
func bulatDesimal(v float64, d int) float64 {
	p := math.Pow(10, float64(d))
	return math.Round(v*p) / p
}
