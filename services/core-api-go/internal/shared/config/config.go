// Package config = membaca pengaturan dari variabel lingkungan / file .env menjadi struct Config.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string // akun core_api (hak minimum), BUKAN core_migrator

	// Token (ADR 0002)
	JWTPrivateKeyFile string // kunci PRIVAT RS256 — RAHASIA, hanya core-api yang memegang
	JWTIssuer         string
	JWTAudience       string
	AccessTTL         time.Duration // umur access token
	RefreshIdleTTL    time.Duration // refresh habis bila tidak dipakai selama ini
	RefreshMaxTTL     time.Duration // batas mutlak sesi sejak login

	// Batas percobaan login
	MaxFailedLogins int
	LockDuration    time.Duration

	// Jadwal laporan (F5): PDF dirender dari halaman cetak frontend oleh Chrome/Edge, dikirim lewat SMTP.
	Reports ReportsConfig
}

type ReportsConfig struct {
	SchedulerEnabled bool   // REPORT_SCHEDULER=false mematikan pengiriman otomatis (kirim manual tetap bisa)
	FrontendURL      string // alamat frontend yang dibuka browser tanpa layar, mis. http://localhost:8881/operans
	BrowserPath      string // kosong = cari Chrome/Edge otomatis
	SMTPHost         string // kosong = email belum dikonfigurasi (pengiriman dicatat gagal)
	SMTPPort         int
	SMTPUser         string
	SMTPPassword     string
	SMTPFrom         string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return def
}

// getdur membaca durasi Go: "15m", "168h", dst.
func getdur(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return d
	}
	return def
}

func Load() (Config, error) {
	loadDotEnv(".env")
	cfg := Config{
		HTTPAddr:          getenv("HTTP_ADDR", ":3346"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTPrivateKeyFile: os.Getenv("JWT_PRIVATE_KEY_FILE"),
		JWTIssuer:         getenv("JWT_ISSUER", "simha-core"),
		JWTAudience:       getenv("JWT_AUDIENCE", "simha-api"),
		AccessTTL:         getdur("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshIdleTTL:    getdur("REFRESH_IDLE_TTL", 7*24*time.Hour),
		RefreshMaxTTL:     getdur("REFRESH_MAX_TTL", 30*24*time.Hour),
		MaxFailedLogins:   getint("MAX_FAILED_LOGINS", 5),
		LockDuration:      getdur("LOGIN_LOCK_DURATION", 15*time.Minute),
		Reports: ReportsConfig{
			SchedulerEnabled: getenv("REPORT_SCHEDULER", "true") != "false",
			FrontendURL:      getenv("FRONTEND_URL", "http://localhost:8881/operans"),
			BrowserPath:      os.Getenv("PDF_BROWSER_PATH"),
			SMTPHost:         os.Getenv("SMTP_HOST"),
			SMTPPort:         getint("SMTP_PORT", 587),
			SMTPUser:         os.Getenv("SMTP_USER"),
			SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
			SMTPFrom:         os.Getenv("SMTP_FROM"),
		},
	}
	if cfg.DatabaseURL == "" || cfg.JWTPrivateKeyFile == "" {
		return cfg, errors.New("DATABASE_URL dan JWT_PRIVATE_KEY_FILE wajib diisi")
	}
	return cfg, nil
}
