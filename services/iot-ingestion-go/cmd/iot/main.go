// Program iot = layanan IoT Ingestion SIMHA (HTTP :3347, TCP Teltonika :5550, WebSocket /ws/live).
//
// Di file ini SEMUA fitur dirakit (ADR 0011): buat repository → service → handler, lalu daftarkan route.
// Siapa memakai siapa:
//
//	ingest        ← device (kenali alat dari IMEI / API key), livestate (kondisi terakhir + kunci per unit),
//	                livestate/ws Hub (siaran WebSocket), masterdata (zona/shift/operator/kalibrasi/ambang alert),
//	                notification (antrian WhatsApp saat alert dibuka)
//	livestate     ← ws Hub (siaran saat unit ditandai offline)
//	notification  ← masterdata (nomor HP penerima, kode unit, nama & zona waktu project)
//	telemetry, alert, geofence, fuel, device: hanya membaca database (tidak memakai fitur lain)
//
// Semua pemakaian lintas fitur lewat interface yang ditulis di sisi PEMAKAI (mis. ingest/service.Enricher).
//
// Yang berjalan bersamaan (errgroup): server HTTP, listener TCP, pengirim WhatsApp, pemeriksa offline,
// pendengar perubahan data induk, dan penutup WebSocket saat berhenti.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	alerthttp "simha/iot-ingestion/internal/alert/delivery/http"
	alertrepo "simha/iot-ingestion/internal/alert/repository"
	alertsvc "simha/iot-ingestion/internal/alert/service"
	devicehttp "simha/iot-ingestion/internal/device/delivery/http"
	devicerepo "simha/iot-ingestion/internal/device/repository"
	devicesvc "simha/iot-ingestion/internal/device/service"
	fuelhttp "simha/iot-ingestion/internal/fuel/delivery/http"
	fuelrepo "simha/iot-ingestion/internal/fuel/repository"
	fuelsvc "simha/iot-ingestion/internal/fuel/service"
	geofencehttp "simha/iot-ingestion/internal/geofence/delivery/http"
	geofencerepo "simha/iot-ingestion/internal/geofence/repository"
	geofencesvc "simha/iot-ingestion/internal/geofence/service"
	ingesthttp "simha/iot-ingestion/internal/ingest/delivery/http"
	ingesttcp "simha/iot-ingestion/internal/ingest/delivery/tcp"
	ingestrepo "simha/iot-ingestion/internal/ingest/repository"
	ingestsvc "simha/iot-ingestion/internal/ingest/service"
	livestatehttp "simha/iot-ingestion/internal/livestate/delivery/http"
	livestatews "simha/iot-ingestion/internal/livestate/delivery/ws"
	livestaterepo "simha/iot-ingestion/internal/livestate/repository"
	livestatesvc "simha/iot-ingestion/internal/livestate/service"
	notificationrepo "simha/iot-ingestion/internal/notification/repository"
	notificationsvc "simha/iot-ingestion/internal/notification/service"
	"simha/iot-ingestion/internal/shared/config"
	"simha/iot-ingestion/internal/shared/database"
	"simha/iot-ingestion/internal/shared/masterdata"
	"simha/iot-ingestion/internal/shared/middleware"
	telemetryhttp "simha/iot-ingestion/internal/telemetry/delivery/http"
	telemetryrepo "simha/iot-ingestion/internal/telemetry/repository"
	telemetrysvc "simha/iot-ingestion/internal/telemetry/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil))) // log terstruktur JSON

	if err := run(); err != nil {
		slog.Error("server berhenti karena error", "err", err)
		os.Exit(1)
	}
}

// run berisi seluruh program. Dipisah dari main agar semua `defer` (tutup database, dll.)
// tetap berjalan: os.Exit di main melewati defer, tetapi return dari run tidak.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("konfigurasi: %w", err)
	}
	// Gin membaca GIN_MODE saat program mulai, sebelum .env dimuat → atur ulang dari nilai .env
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}

	pubKey, err := middleware.LoadPublicKey(cfg.JWTPublicKeyFile)
	if err != nil {
		return fmt.Errorf("kunci publik JWT (%s): %w", cfg.JWTPublicKeyFile, err)
	}

	// ctx otomatis "dibatalkan" saat Ctrl+C (Interrupt) atau SIGTERM (dari Docker/systemd)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	slog.Info("database terhubung")

	// --- livestate: kondisi terakhir semua equipment dibaca SEKALI dari database, lalu dipegang di memori (ADR 0005)
	liveStates := livestatesvc.New(livestaterepo.New(pool))
	n, err := liveStates.Load(ctx)
	if err != nil {
		return fmt.Errorf("muat kondisi terakhir equipment: %w", err)
	}
	slog.Info("kondisi terakhir equipment dimuat", "jumlah", n)

	// Penyiar WebSocket: setiap perubahan kondisi equipment dikirim ke browser project-nya
	hub := livestatews.NewHub(cfg.WSMaxConnections, 64)

	// --- masterdata: data induk core (shift, equipment, operator, breakdown) di memori, diperbarui lewat NOTIFY
	// (ADR 0009). Gagal dimuat bukan alasan berhenti: telemetry tetap disimpan, kolom pengayaan kosong sampai berhasil.
	master := masterdata.New(masterdata.NewRepository(pool))
	if err := master.Load(ctx); err != nil {
		slog.Error("data induk core belum bisa dimuat; dicoba lagi otomatis", "err", err)
	}
	st := master.Status()
	slog.Info("data induk dimuat", "project", st.Projects, "equipment", st.Equipment, "operator", st.Operators, "breakdown", st.Breakdowns, "zona", st.Zones)
	listener := masterdata.NewListener(cfg.DatabaseURL, "simha_master_changed")
	defer listener.Close()

	// --- notification: notifikasi WhatsApp alert lewat antrian iot.notifications (nonaktif bila WHATSAPP_API_URL kosong)
	notifier := notificationsvc.New(notificationsvc.Config{URL: cfg.WhatsAppURL, Timeout: cfg.WhatsAppTimeout,
		MaxAttempts: cfg.WhatsAppMaxAttempts}, notificationrepo.New(pool), master)

	// --- device: alat GPS (dipakai halaman Device dan untuk mengenali alat yang mengirim data)
	devices := devicesvc.New(devicerepo.New(pool))

	// --- ingest: SATU service, dipakai bersama pintu HTTP dan TCP
	ingest := ingestsvc.NewIngest(ingestrepo.New(pool), liveStates, hub).WithEnricher(master).WithNotifier(notifier)

	// --- fitur baca dashboard
	telemetry := telemetrysvc.New(telemetryrepo.New(pool))
	alerts := alertsvc.New(alertrepo.New(pool))
	geofences := geofencesvc.New(geofencerepo.New(pool))
	fuels := fuelsvc.New(fuelrepo.New(pool))

	// --- route HTTP
	jwtOpt := middleware.JWTOptions{PublicKey: pubKey, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())

	r.GET("/health", health(pool))

	// Peta live → WebSocket; login lewat pesan pertama (browser tidak bisa memasang header Authorization)
	r.GET("/ws/live", livestatews.LiveSocket(livestatews.LiveSocketConfig{
		Hub:            hub,
		Verifier:       middleware.NewJWTVerifier(jwtOpt),
		AllowedOrigins: cfg.WSAllowedOrigins,
	}))

	// Alat/simulator mengirim data → butuh X-API-Key (dipasang oleh handler ingest, bukan token pengguna)
	ingesthttp.New(ingest, devices).Register(r.Group("/api/v1"))

	// Endpoint baca untuk pengguna/dashboard → butuh access token SIMHA (RS256, ADR 0002)
	api := r.Group("/api/v1", middleware.JWTAuth(jwtOpt))
	livestatehttp.New(liveStates).Register(api)
	telemetryhttp.New(telemetry, time.Duration(cfg.MaxHistoryHours)*time.Hour, cfg.MaxHistoryRows).Register(api)
	alerthttp.New(alerts).Register(api)
	geofencehttp.New(geofences).Register(api)
	fuelhttp.New(fuels).Register(api)
	devicehttp.New(devices).Register(api) // Kelola alat (System Data → Device): admin project & superadmin

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second, // klien lambat tidak boleh menahan koneksi
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	tcpListener := &ingesttcp.Listener{
		Addr:    cfg.TCPAddr,
		Devices: devices,
		Ingest:  ingest,
		LogRaw:  cfg.TCPLogRaw,
		IO:      ingesttcp.IOSettings{FuelPercent: uint16(cfg.FuelPercentIOID)},
	}

	// errgroup menjalankan HTTP dan TCP bersamaan. Bila salah satu gagal (mis. port dipakai),
	// gctx dibatalkan sehingga yang lain ikut berhenti dengan rapi.
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return serveHTTP(gctx, srv) })
	g.Go(func() error { return tcpListener.Run(gctx) })
	g.Go(func() error { // pengirim WhatsApp: cek antrian tiap 5 detik (atau segera saat ada alert baru)
		notifier.Run(gctx, 5*time.Second)
		return nil
	})
	g.Go(func() error { // offline otomatis: unit tanpa data ≥ OFFLINE_AFTER_MINUTES
		livestatesvc.WatchOffline(gctx, liveStates, hub, cfg.OfflineAfter)
		return nil
	})
	// breakdown / operator berubah (input, ubah, unggah, hapus) → kondisi terakhir unit ikut berubah seketika
	refreshOps := make(chan struct{}, 1)
	master.OnReload(func(parts []string) {
		for _, p := range parts {
			if p == masterdata.PartBreakdowns || p == masterdata.PartOperators || p == masterdata.PartProjects || p == masterdata.PartEquipment {
				select {
				case refreshOps <- struct{}{}:
				default: // sudah ada sinyal menunggu
				}
				return
			}
		}
	})
	g.Go(func() error { // sinkron breakdown & operator ke kondisi terakhir (segera saat berubah, dan tiap menit untuk perubahan karena waktu)
		livestatesvc.WatchOperational(gctx, liveStates, masterOperational{master}, hub, refreshOps, time.Minute)
		return nil
	})
	g.Go(func() error { // dengarkan perubahan data induk dari core
		master.Run(gctx, listener, cfg.MasterRefresh, 500*time.Millisecond)
		return nil
	})
	g.Go(func() error { // saat berhenti: putus semua WebSocket dengan kode "server berhenti"
		<-gctx.Done()
		hub.Close()
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("server berhenti dengan rapi")
	return nil
}

// health = GET /health: "sehat" hanya bila database juga bisa dijangkau.
func health(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "up"})
	}
}

// serveHTTP menjalankan server HTTP sampai ctx dibatalkan, lalu graceful shutdown (maks. 10 detik).
func serveHTTP(ctx context.Context, srv *http.Server) error {
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP mendengarkan", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		slog.Info("sinyal berhenti diterima, menutup HTTP")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// masterOperational menyambungkan data induk (masterdata) ke pembaruan kondisi terakhir (livestate), yang sengaja tidak
// saling mengimpor.
type masterOperational struct{ m *masterdata.Cache }

func (a masterOperational) OperationalAt(projectID, equipmentID *uuid.UUID, t time.Time) (livestatesvc.Operational, bool) {
	op, ok := a.m.OperationalAt(projectID, equipmentID, t)
	return livestatesvc.Operational{
		BreakdownKnown: op.BreakdownKnown, IsBreakdown: op.IsBreakdown, BreakdownDescription: op.BreakdownDescription,
		OperatorKnown: op.OperatorKnown, Operator: op.Operator,
	}, ok
}
