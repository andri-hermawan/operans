// core-api = layanan data induk & input harian SIMHA (login, user, project, equipment, data harian,
// peta, aturan alert, laporan).
//
// File ini satu-satunya tempat SEMUA fitur dirakit. Setiap fitur (folder internal/<fitur>) dibuat dengan
// urutan yang sama: repository (SQL) → service (aturan) → delivery/http (handler + daftar route).
//
// Siapa memakai siapa (fitur memakai fitur lain HANYA lewat service-nya, via interface kecil):
//
//	auth            → project (superadmin membuka project mana pun)
//	user            → auth (cari email, ganti password, cabut sesi), project
//	project         → company (company_id ada?)
//	shift           → project
//	equipment       → project
//	activation      → equipment
//	fuelcalibration → equipment
//	operator, breakdown, weighbridge
//	                → dailyinput.Loader = project + shift + equipment + reference
//	importexcel     → dailyinput.Loader + operator, activation, breakdown, weighbridge (pengurai baris Excel)
//	maparea         → project
//	alertrule       → project, alerttype, maparea (nama zona)
//	reportschedule  → shift, token (halaman cetak PDF)
//	company, reference, alerttype, report → tidak memakai fitur lain
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
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	activationhttp "simha/core-api/internal/activation/delivery/http"
	activationrepo "simha/core-api/internal/activation/repository"
	activationsvc "simha/core-api/internal/activation/service"
	alertrulehttp "simha/core-api/internal/alertrule/delivery/http"
	alertrulerepo "simha/core-api/internal/alertrule/repository"
	alertrulesvc "simha/core-api/internal/alertrule/service"
	alerttypehttp "simha/core-api/internal/alerttype/delivery/http"
	alerttyperepo "simha/core-api/internal/alerttype/repository"
	alerttypesvc "simha/core-api/internal/alerttype/service"
	authhttp "simha/core-api/internal/auth/delivery/http"
	authrepo "simha/core-api/internal/auth/repository"
	authsvc "simha/core-api/internal/auth/service"
	breakdownhttp "simha/core-api/internal/breakdown/delivery/http"
	breakdownrepo "simha/core-api/internal/breakdown/repository"
	breakdownsvc "simha/core-api/internal/breakdown/service"
	companyhttp "simha/core-api/internal/company/delivery/http"
	companyrepo "simha/core-api/internal/company/repository"
	companysvc "simha/core-api/internal/company/service"
	equipmenthttp "simha/core-api/internal/equipment/delivery/http"
	equipmentrepo "simha/core-api/internal/equipment/repository"
	equipmentsvc "simha/core-api/internal/equipment/service"
	fuelcalibrationhttp "simha/core-api/internal/fuelcalibration/delivery/http"
	fuelcalibrationrepo "simha/core-api/internal/fuelcalibration/repository"
	fuelcalibrationsvc "simha/core-api/internal/fuelcalibration/service"
	importexcelhttp "simha/core-api/internal/importexcel/delivery/http"
	importexcelsvc "simha/core-api/internal/importexcel/service"
	mapareahttp "simha/core-api/internal/maparea/delivery/http"
	maparearepo "simha/core-api/internal/maparea/repository"
	mapareasvc "simha/core-api/internal/maparea/service"
	monthlyplanhttp "simha/core-api/internal/monthlyplan/delivery/http"
	monthlyplanrepo "simha/core-api/internal/monthlyplan/repository"
	monthlyplansvc "simha/core-api/internal/monthlyplan/service"
	operatorhttp "simha/core-api/internal/operator/delivery/http"
	operatorrepo "simha/core-api/internal/operator/repository"
	operatorsvc "simha/core-api/internal/operator/service"
	projecthttp "simha/core-api/internal/project/delivery/http"
	projectrepo "simha/core-api/internal/project/repository"
	projectsvc "simha/core-api/internal/project/service"
	rainslipperyhttp "simha/core-api/internal/rainslippery/delivery/http"
	rainslipperyrepo "simha/core-api/internal/rainslippery/repository"
	rainslipperysvc "simha/core-api/internal/rainslippery/service"
	referencehttp "simha/core-api/internal/reference/delivery/http"
	referencerepo "simha/core-api/internal/reference/repository"
	referencesvc "simha/core-api/internal/reference/service"
	reporthttp "simha/core-api/internal/report/delivery/http"
	reportrepo "simha/core-api/internal/report/repository"
	reportsvc "simha/core-api/internal/report/service"
	reportschedulehttp "simha/core-api/internal/reportschedule/delivery/http"
	reportschedulerepo "simha/core-api/internal/reportschedule/repository"
	reportschedulesvc "simha/core-api/internal/reportschedule/service"
	ritasehttp "simha/core-api/internal/ritase/delivery/http"
	ritaserepo "simha/core-api/internal/ritase/repository"
	ritasesvc "simha/core-api/internal/ritase/service"
	"simha/core-api/internal/shared/config"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/shared/middleware"
	"simha/core-api/internal/shared/token"
	shifthttp "simha/core-api/internal/shift/delivery/http"
	shiftrepo "simha/core-api/internal/shift/repository"
	shiftsvc "simha/core-api/internal/shift/service"
	stockhttp "simha/core-api/internal/stockemplacement/delivery/http"
	stockrepo "simha/core-api/internal/stockemplacement/repository"
	stocksvc "simha/core-api/internal/stockemplacement/service"
	timesheethttp "simha/core-api/internal/timesheet/delivery/http"
	timesheetrepo "simha/core-api/internal/timesheet/repository"
	timesheetsvc "simha/core-api/internal/timesheet/service"
	userhttp "simha/core-api/internal/user/delivery/http"
	userrepo "simha/core-api/internal/user/repository"
	usersvc "simha/core-api/internal/user/service"
	weighbridgehttp "simha/core-api/internal/weighbridge/delivery/http"
	weighbridgerepo "simha/core-api/internal/weighbridge/repository"
	weighbridgesvc "simha/core-api/internal/weighbridge/service"
	workhourhttp "simha/core-api/internal/workhour/delivery/http"
	workhourrepo "simha/core-api/internal/workhour/repository"
	workhoursvc "simha/core-api/internal/workhour/service"
	yearlyplanhttp "simha/core-api/internal/yearlyplan/delivery/http"
	yearlyplanrepo "simha/core-api/internal/yearlyplan/repository"
	yearlyplansvc "simha/core-api/internal/yearlyplan/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil))) // log terstruktur JSON

	if err := run(); err != nil {
		slog.Error("server berhenti karena error", "err", err)
		os.Exit(1)
	}
}

// run berisi seluruh program. Dipisah dari main agar semua `defer` tetap berjalan
// (os.Exit di main melewati defer, tetapi return dari run tidak).
func run() error {
	// 1. Konfigurasi (.env)
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("konfigurasi: %w", err)
	}
	// Gin membaca GIN_MODE saat program mulai, sebelum .env dimuat → atur ulang dari nilai .env
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}

	// 2. Kunci JWT (membuat & memeriksa access token)
	privKey, err := token.LoadPrivateKey(cfg.JWTPrivateKeyFile)
	if err != nil {
		return fmt.Errorf("kunci privat JWT (%s): %w", cfg.JWTPrivateKeyFile, err)
	}
	signer, err := token.NewSigner(privKey, cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTTL)
	if err != nil {
		return fmt.Errorf("signer JWT: %w", err)
	}

	// ctx otomatis "dibatalkan" saat Ctrl+C (Interrupt) atau SIGTERM (dari Docker/systemd)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Database (satu pool untuk semua repository)
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	slog.Info("database terhubung")

	// 4. Rakit fitur: repository → service (urutan: fitur yang dipakai fitur lain dibuat lebih dulu)
	companyService := companysvc.New(companyrepo.New(pool))
	projectService := projectsvc.New(projectrepo.New(pool), companyService)
	authService := authsvc.New(authrepo.New(pool), projectService, signer, authsvc.Policy{
		RefreshIdleTTL:  cfg.RefreshIdleTTL,
		RefreshMaxTTL:   cfg.RefreshMaxTTL,
		MaxFailedLogins: cfg.MaxFailedLogins,
		LockDuration:    cfg.LockDuration,
	})
	userService := usersvc.New(userrepo.New(pool), authService, projectService)
	shiftService := shiftsvc.New(shiftrepo.New(pool), projectService)
	equipmentService := equipmentsvc.New(equipmentrepo.New(pool), projectService)
	activationService := activationsvc.New(activationrepo.New(pool), equipmentService)
	fuelCalibrationService := fuelcalibrationsvc.New(fuelcalibrationrepo.New(pool), equipmentService)
	referenceService := referencesvc.New(referencerepo.New(pool))

	// Input harian memuat project + shift + master equipment + daftar pilihan lewat service masing-masing.
	daily := dailyinput.Loader{
		Projects:   projectService,
		Shifts:     shiftService,
		Equipment:  equipmentService,
		References: referenceService,
	}
	operatorService := operatorsvc.New(operatorrepo.New(pool), daily)
	breakdownService := breakdownsvc.New(breakdownrepo.New(pool), daily)
	weighbridgeService := weighbridgesvc.New(weighbridgerepo.New(pool), daily)
	stockService := stocksvc.New(stockrepo.New(pool), daily)
	timeSheetService := timesheetsvc.New(timesheetrepo.New(pool), daily)
	ritaseService := ritasesvc.New(ritaserepo.New(pool), daily)
	workHourService := workhoursvc.New(workhourrepo.New(pool), daily)
	rainSlipperyService := rainslipperysvc.New(rainslipperyrepo.New(pool), daily)
	yearlyPlanService := yearlyplansvc.New(yearlyplanrepo.New(pool), daily)
	monthlyPlanService := monthlyplansvc.New(monthlyplanrepo.New(pool), daily)
	// Upload Excel: jenis impor (bagian URL /imports/:type) → fitur yang mengurai & menyimpan barisnya.
	importService := importexcelsvc.New(daily, map[string]importexcelsvc.RowImporter{
		"operator-assignments":    operatorService,
		"equipment-activations":   activationService,
		"equipment-breakdowns":    breakdownService,
		"weighbridge-tickets":     weighbridgeService,
		"stock-emplacements":      stockService,
		"yearly-plans":            yearlyPlanService,
		"monthly-plans":           monthlyPlanService,
		"time-sheets":             timeSheetService,
		"ritases":                 ritaseService,
		"work-hour-notifications": workHourService,
		"rain-slippery-logs":      rainSlipperyService,
	})

	mapAreaService := mapareasvc.New(maparearepo.New(pool), projectService)
	alertTypeService := alerttypesvc.New(alerttyperepo.New(pool))
	alertRuleService := alertrulesvc.New(alertrulerepo.New(pool), projectService, alertTypeService, mapAreaService)
	reportService := reportsvc.New(reportrepo.New(pool))
	rc := cfg.Reports
	reportScheduleService := reportschedulesvc.New(reportschedulerepo.New(pool), shiftService, signer, reportschedulesvc.Config{
		FrontendURL: rc.FrontendURL,
		BrowserPath: rc.BrowserPath,
		SMTP:        reportschedulesvc.SMTPConfig{Host: rc.SMTPHost, Port: rc.SMTPPort, User: rc.SMTPUser, Password: rc.SMTPPassword, From: rc.SMTPFrom},
	})

	// 5. Route: URL → handler, beserta middleware yang menjaganya
	r := gin.New()
	r.Use(gin.Recovery(), middleware.RequestID(), middleware.AccessLog())
	r.GET("/health", health(pool))
	authhttp.New(authService, signer).RegisterPublic(r) // /.well-known/jwks.json + /auth/*

	// API data — semua butuh access token; hak akses (superadmin / admin project) dicek di service
	api := r.Group("/api/v1", middleware.RequireAccess(signer))
	userhttp.New(userService).Register(api)
	companyhttp.New(companyService).Register(api)
	projecthttp.New(projectService).Register(api)
	reporthttp.New(reportService).Register(api)
	reportschedulehttp.New(reportScheduleService).Register(api)
	shifthttp.New(shiftService).Register(api)
	equipmenthttp.New(equipmentService).Register(api)
	activationhttp.New(activationService).Register(api)
	referencehttp.New(referenceService).Register(api)
	operatorhttp.New(operatorService).Register(api)
	breakdownhttp.New(breakdownService).Register(api)
	weighbridgehttp.New(weighbridgeService).Register(api)
	stockhttp.New(stockService).Register(api)
	yearlyplanhttp.New(yearlyPlanService).Register(api)
	monthlyplanhttp.New(monthlyPlanService).Register(api)
	timesheethttp.New(timeSheetService).Register(api)
	ritasehttp.New(ritaseService).Register(api)
	workhourhttp.New(workHourService).Register(api)
	rainslipperyhttp.New(rainSlipperyService).Register(api)
	importexcelhttp.New(importService).Register(api)
	mapareahttp.New(mapAreaService).Register(api)
	fuelcalibrationhttp.New(fuelCalibrationService).Register(api)
	alerttypehttp.New(alertTypeService).Register(api)
	alertrulehttp.New(alertRuleService).Register(api)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second, // upload Excel (B5) butuh waktu baca lebih panjang
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 6. Jalankan server HTTP (+ penjadwal laporan) sampai ada sinyal berhenti
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return serveHTTP(gctx, srv) })
	if rc.SchedulerEnabled {
		g.Go(func() error { reportScheduleService.StartScheduler(gctx); return nil })
		slog.Info("jadwal laporan aktif", "frontend", rc.FrontendURL, "smtp", rc.SMTPHost != "")
	}
	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("server berhenti dengan rapi")
	return nil
}

// health: "sehat" hanya bila database juga bisa dijangkau.
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
