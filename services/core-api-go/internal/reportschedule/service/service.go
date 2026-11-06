// Package service (fitur reportschedule) = Report Schedule: jadwal kirim laporan (F5).
// Lihat & ubah: admin project & superadmin. Penjadwal (runner.go) memeriksa jadwal tiap menit, membuat PDF
// dari halaman cetak frontend dengan Chrome/Edge tanpa layar (pdf.go), lalu mengirim email (mail.go).
package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/reportschedule/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/token"
	shiftdomain "simha/core-api/internal/shift/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectID *uuid.UUID, activeOnly bool) ([]domain.ReportSchedule, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.ReportSchedule, error)
	Insert(ctx context.Context, s domain.ReportSchedule, by uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, s domain.ReportSchedule, by uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	HasScheduledRun(ctx context.Context, id uuid.UUID, runDate string) (bool, error)
	InsertLog(ctx context.Context, id uuid.UUID, l domain.ReportScheduleLog) error
	ListLogs(ctx context.Context, id uuid.UUID) ([]domain.ReportScheduleLog, error)
}

// ShiftLister = fitur shift (memeriksa shift_code jadwal).
type ShiftLister interface {
	ListActive(ctx context.Context, projectID uuid.UUID) ([]shiftdomain.Shift, error)
}

// TokenIssuer = pembuat access token untuk browser tanpa layar yang membuka halaman cetak.
type TokenIssuer interface {
	IssueAccess(sub token.AccessSubject) (string, time.Time, error)
}

// Config = pengaturan pembuatan PDF & pengiriman email.
type Config struct {
	FrontendURL string
	BrowserPath string
	SMTP        SMTPConfig
}

type Service struct {
	repo   Repository
	shifts ShiftLister
	tokens TokenIssuer
	cfg    Config
	// dapat diganti di test
	render func(ctx context.Context, url string) ([]byte, error)
	send   func(to []string, subject, html, fileName string, pdf []byte) error
	now    func() time.Time
}

func New(repo Repository, shifts ShiftLister, tokens TokenIssuer, cfg Config) *Service {
	r := &Service{repo: repo, shifts: shifts, tokens: tokens, cfg: cfg, now: time.Now}
	r.render = func(ctx context.Context, url string) ([]byte, error) { return renderPDF(ctx, cfg.BrowserPath, url) }
	r.send = func(to []string, subject, html, fileName string, pdf []byte) error {
		return sendMail(cfg.SMTP, to, subject, html, fileName, pdf)
	}
	return r
}

var ymd = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ---------------------------------------------------------------------
// Jadwal laporan — lihat & ubah: admin project & superadmin
// ---------------------------------------------------------------------

func (s *Service) List(ctx context.Context, a access.Actor, projectFilter *uuid.UUID) ([]domain.ReportSchedule, error) {
	pid := projectFilter
	if pid == nil && !a.SuperAdmin {
		pid = a.ProjectID
	}
	if pid != nil && !a.CanManageProject(*pid) || pid == nil && !a.SuperAdmin {
		return nil, apperror.ErrForbidden
	}
	return s.repo.List(ctx, pid, false)
}

// schedule = jadwal yang boleh dikelola pemanggil.
func (s *Service) schedule(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.ReportSchedule, error) {
	sc, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if sc == nil || !a.CanView(sc.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	if !a.CanManageProject(sc.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	return sc, nil
}

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Service) Save(ctx context.Context, a access.Actor, id *uuid.UUID, in domain.ReportScheduleInput) (*domain.ReportSchedule, error) {
	var sc domain.ReportSchedule
	if id == nil {
		pid := in.ProjectID
		if !a.SuperAdmin || pid == nil {
			pid = a.ProjectID
		}
		if pid == nil {
			return nil, apperror.Invalid("project_id wajib diisi")
		}
		if !a.CanManageProject(*pid) {
			return nil, apperror.ErrForbidden
		}
		sc = domain.ReportSchedule{ProjectID: *pid, DateOffset: -1, SendTime: "09:00", IsActive: true, ReportType: "alert_summary"}
	} else {
		cur, err := s.schedule(ctx, a, *id)
		if err != nil {
			return nil, err
		}
		sc = *cur
	}

	if in.Name != nil {
		sc.Name = strings.TrimSpace(*in.Name)
	}
	if in.ReportType != nil {
		sc.ReportType = *in.ReportType
	}
	if in.ShiftCode != nil {
		if v := strings.TrimSpace(*in.ShiftCode); v == "" {
			sc.ShiftCode = nil
		} else {
			sc.ShiftCode = &v
		}
	}
	if in.DateOffset != nil {
		sc.DateOffset = *in.DateOffset
	}
	if in.SendTime != nil {
		sc.SendTime = *in.SendTime
	}
	if in.RecipientUserIDs != nil {
		sc.RecipientUserIDs = *in.RecipientUserIDs
	}
	if in.IsActive != nil {
		sc.IsActive = *in.IsActive
	}

	switch {
	case sc.Name == "" || utf8.RuneCountInString(sc.Name) > 150:
		return nil, apperror.Invalid("name wajib diisi (maks. 150 karakter)")
	case domain.ReportTypes[sc.ReportType].Label == "":
		return nil, apperror.Invalid("report_type tidak dikenal")
	case sc.DateOffset != -1 && sc.DateOffset != 0:
		return nil, apperror.Invalid("date_offset hanya -1 (kemarin) atau 0 (hari ini)")
	case !hhmmRe.MatchString(sc.SendTime):
		return nil, apperror.Invalid("send_time format HH:MM")
	case len(sc.RecipientUserIDs) == 0:
		return nil, apperror.Invalid("pilih minimal satu penerima")
	}
	if sc.ShiftCode != nil {
		shifts, err := s.shifts.ListActive(ctx, sc.ProjectID)
		if err != nil {
			return nil, err
		}
		if _, ok := shiftdomain.ResolveShift(shifts, *sc.ShiftCode); !ok {
			return nil, apperror.Invalid("shift %s tidak ada di project ini", *sc.ShiftCode)
		}
	}

	if id == nil {
		newID, err := s.repo.Insert(ctx, sc, a.UserID)
		if err != nil {
			return nil, err
		}
		id = &newID
	} else if err := s.repo.Update(ctx, sc, a.UserID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, *id)
}

func (s *Service) Delete(ctx context.Context, a access.Actor, id uuid.UUID) error {
	if _, err := s.schedule(ctx, a, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Logs(ctx context.Context, a access.Actor, id uuid.UUID) ([]domain.ReportScheduleLog, error) {
	if _, err := s.schedule(ctx, a, id); err != nil {
		return nil, err
	}
	return s.repo.ListLogs(ctx, id)
}
