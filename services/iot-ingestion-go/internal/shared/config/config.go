// Package config membaca pengaturan layanan dari variabel lingkungan (.env) menjadi satu struct Config.
// Semua nama variabel (HTTP_ADDR, DATABASE_URL, ...) hanya didefinisikan di sini.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr            string
	TCPAddr             string
	DatabaseURL         string
	JWTPublicKeyFile    string // kunci PUBLIK RS256 dari core-api (layanan ini tidak bisa membuat token)
	JWTIssuer           string
	JWTAudience         string
	MaxHistoryHours     int
	MaxHistoryRows      int
	TCPLogRaw           bool          // catat paket TCP mentah (hex) — untuk menyetel alat baru
	WSAllowedOrigins    []string      // asal halaman yang boleh membuka WebSocket, mis. "localhost:8881"
	WSMaxConnections    int           // batas koneksi WebSocket bersamaan
	MasterRefresh       time.Duration // muat ulang penuh data induk core bila tidak ada NOTIFY (cadangan)
	WhatsAppURL         string        // server WhatsApp (sama dengan sistem lama); kosong = nonaktif
	WhatsAppTimeout     time.Duration // batas waktu satu kiriman
	WhatsAppMaxAttempts int           // batas percobaan sebelum ditandai gagal
	OfflineAfter        time.Duration // tanpa data selama ini → status offline di peta live
	FuelPercentIOID     int           // ID AVL persen level tangki dari ECU (unit bertipe percentage); 0 = tidak dibaca
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// getlist membaca daftar dipisah koma: "a, b,c" → [a b c].
func getlist(key, def string) []string {
	var out []string
	for _, v := range strings.Split(getenv(key, def), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func Load() (Config, error) {
	loadDotEnv(".env")
	cfg := Config{
		HTTPAddr:            getenv("HTTP_ADDR", ":5550"),
		TCPAddr:             getenv("TCP_ADDR", ":3347"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTPublicKeyFile:    os.Getenv("JWT_PUBLIC_KEY_FILE"),
		JWTIssuer:           getenv("JWT_ISSUER", "simha-core"),
		JWTAudience:         getenv("JWT_AUDIENCE", "simha-api"),
		MaxHistoryHours:     getint("MAX_HISTORY_RANGE_HOURS", 24),
		MaxHistoryRows:      getint("MAX_HISTORY_ROWS", 1000),
		TCPLogRaw:           os.Getenv("TCP_LOG_RAW") == "true",
		WSAllowedOrigins:    getlist("WS_ALLOWED_ORIGINS", "localhost:*,127.0.0.1:*"),
		WSMaxConnections:    getint("WS_MAX_CONNECTIONS", 500),
		MasterRefresh:       time.Duration(getint("MASTERDATA_REFRESH_MINUTES", 10)) * time.Minute,
		WhatsAppURL:         os.Getenv("WHATSAPP_API_URL"),
		WhatsAppTimeout:     time.Duration(getint("WHATSAPP_TIMEOUT_MS", 10000)) * time.Millisecond,
		WhatsAppMaxAttempts: getint("WHATSAPP_MAX_ATTEMPTS", 3),
		OfflineAfter:        time.Duration(getint("OFFLINE_AFTER_MINUTES", 10)) * time.Minute,
		FuelPercentIOID:     getint("FUEL_PERCENT_IO_ID", 89),
	}
	if cfg.DatabaseURL == "" || cfg.JWTPublicKeyFile == "" {
		return cfg, errors.New("DATABASE_URL dan JWT_PUBLIC_KEY_FILE wajib diisi")
	}
	return cfg, nil
}
