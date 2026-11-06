package config

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv membaca file .env (bila ada) di folder tempat program dijalankan, agar cukup
// `go run ./cmd/iot` dari cmd/PowerShell tanpa memuat .env secara manual.
// Variabel yang SUDAH diset di sistem/terminal tidak ditimpa (mis. di Docker/produksi).
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // tidak ada .env = pakai variabel sistem saja
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.Trim(strings.TrimSpace(val), `"'`)
		if _, set := os.LookupEnv(key); !set && key != "" {
			os.Setenv(key, val)
		}
	}
}
