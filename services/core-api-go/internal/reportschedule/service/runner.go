package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/reportschedule/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/token"
)

// Jadwal yang terlewat (mis. server baru dinyalakan) masih dikejar dalam jendela ini (sama dengan sistem lama).
const catchUpWindow = 120 * time.Minute

var runMu sync.Mutex // satu laporan dirender pada satu waktu (browser tanpa layar cukup berat)

// StartScheduler memeriksa jadwal tiap menit sampai ctx selesai. Jam kirim dibaca dari database,
// jadi perubahan di halaman Report Schedule langsung berlaku.
func (s *Service) StartScheduler(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		s.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	list, err := s.repo.List(ctx, nil, true)
	if err != nil {
		slog.Error("jadwal laporan: baca gagal", "err", err)
		return
	}
	for _, sc := range list {
		if due, runDate := s.isDue(sc); due {
			done, err := s.repo.HasScheduledRun(ctx, sc.ID, runDate)
			if err != nil || done {
				continue
			}
			if _, err := s.run(ctx, sc, "schedule", ""); err != nil {
				slog.Warn("jadwal laporan gagal", "schedule_id", sc.ID, "err", err) // sudah dicatat di log jadwal
			}
		}
	}
}

// isDue: jam lokal project sudah lewat jam kirim, tetapi belum lebih dari jendela kejar.
func (s *Service) isDue(sc domain.ReportSchedule) (bool, string) {
	loc, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	now := s.now().In(loc)
	send, _ := time.ParseInLocation("2006-01-02 15:04", now.Format("2006-01-02")+" "+sc.SendTime, loc)
	late := now.Sub(send)
	return late >= 0 && late <= catchUpWindow, now.Format("2006-01-02")
}

// SendNow mengirim satu jadwal sekarang (tombol "Send Now"). reportDate kosong = sesuai date_offset.
func (s *Service) SendNow(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.ReportScheduleLog, error) {
	sc, err := s.schedule(ctx, a, id)
	if err != nil {
		return nil, err
	}
	return s.run(ctx, *sc, "manual", "")
}

// Preview membuat PDF tanpa mengirim email (cek tampilan / bila SMTP belum diatur).
func (s *Service) Preview(ctx context.Context, a access.Actor, id uuid.UUID, reportDate string) ([]byte, string, error) {
	sc, err := s.schedule(ctx, a, id)
	if err != nil {
		return nil, "", err
	}
	date := reportDate
	if date == "" {
		date = s.reportDate(*sc)
	} else if !ymd.MatchString(date) {
		return nil, "", apperror.Invalid("date format YYYY-MM-DD")
	}
	pdf, err := s.pdf(ctx, *sc, date)
	return pdf, fileName(*sc, date), err
}

func (s *Service) reportDate(sc domain.ReportSchedule) string {
	loc, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return s.now().In(loc).AddDate(0, 0, sc.DateOffset).Format("2006-01-02")
}

func fileName(sc domain.ReportSchedule, date string) string {
	shift := "harian"
	if sc.ShiftCode != nil {
		shift = strings.ToLower(*sc.ShiftCode)
	}
	return fmt.Sprintf("%s_%s_%s.pdf", sc.ReportType, date, shift)
}

// pdf membuka halaman cetak frontend dengan token sementara (di bagian #, tidak terkirim ke server/log),
// menunggu halaman siap (window.__REPORT_READY__), lalu mencetak ke PDF.
func (s *Service) pdf(ctx context.Context, sc domain.ReportSchedule, date string) ([]byte, error) {
	owner := sc.UpdatedBy
	if owner == nil {
		owner = sc.CreatedBy
	}
	if owner == nil {
		return nil, errors.New("jadwal tanpa pembuat; simpan ulang jadwal")
	}
	pid := sc.ProjectID
	token, _, err := s.tokens.IssueAccess(token.AccessSubject{UserID: *owner, Name: "Report Schedule", ProjectID: &pid, ProjectRole: access.RoleViewer})
	if err != nil {
		return nil, err
	}
	q := url.Values{"date": {date}}
	if sc.ShiftCode != nil {
		q.Set("shift", *sc.ShiftCode)
	}
	u := strings.TrimRight(s.cfg.FrontendURL, "/") + domain.ReportTypes[sc.ReportType].PrintPath + "?" + q.Encode() + "#token=" + url.QueryEscape(token)
	runMu.Lock()
	defer runMu.Unlock()
	return s.render(ctx, u)
}

// run membuat PDF, mengirim ke penerima aktif, dan SELALU mencatat hasilnya di log jadwal.
func (s *Service) run(ctx context.Context, sc domain.ReportSchedule, trigger, reportDate string) (*domain.ReportScheduleLog, error) {
	if reportDate == "" {
		reportDate = s.reportDate(sc)
	}
	_, runDate := s.isDue(sc)
	l := domain.ReportScheduleLog{RunDate: runDate, ReportDate: reportDate, Trigger: trigger, Status: "sent", Recipients: len(sc.Recipients)}

	err := func() error {
		if len(sc.Recipients) == 0 {
			return errors.New("tidak ada penerima aktif")
		}
		if s.cfg.SMTP.Host == "" {
			return errors.New("SMTP belum dikonfigurasi (isi SMTP_HOST di .env core-api)")
		}
		pdf, err := s.pdf(ctx, sc, reportDate)
		if err != nil {
			return fmt.Errorf("buat PDF: %w", err)
		}
		to := make([]string, 0, len(sc.Recipients))
		for _, r := range sc.Recipients {
			to = append(to, r.Email)
		}
		label := domain.ReportTypes[sc.ReportType].Label
		shift := "satu hari"
		if sc.ShiftCode != nil {
			shift = *sc.ShiftCode
		}
		subject := fmt.Sprintf("[SIMHA] %s %s (%s)", label, reportDate, shift)
		html := fmt.Sprintf("<p>Terlampir laporan <b>%s</b> tanggal operasional <b>%s</b> (%s).</p><p>Email otomatis dari jadwal \"%s\" — SIMHA.</p>",
			label, reportDate, shift, sc.Name)
		return s.send(to, subject, html, fileName(sc, reportDate), pdf)
	}()
	if err != nil {
		msg := err.Error()
		l.Status, l.ErrorMessage = "failed", &msg
	}
	if lerr := s.repo.InsertLog(ctx, sc.ID, l); lerr != nil {
		slog.Error("catat log jadwal gagal", "schedule_id", sc.ID, "err", lerr)
	}
	return &l, err
}
