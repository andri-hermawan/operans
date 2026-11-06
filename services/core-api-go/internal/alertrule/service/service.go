// Package service (fitur alertrule) = Alert Rule: aturan alert per project / zona + penerima WhatsApp
// (B7, ADR 0008). Lihat = semua role project; ubah = admin project. Jenis alert dicek lewat fitur
// alerttype, nama zona lewat fitur maparea, project lewat fitur project.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/alertrule/domain"
	alerttypedomain "simha/core-api/internal/alerttype/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.AlertRule, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error)
	Save(ctx context.Context, a domain.AlertRule, by uuid.UUID) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Effective(ctx context.Context, projectID uuid.UUID, zoneName string) ([]domain.EffectiveAlertRule, error)
	ProjectMembers(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.AlertRecipient, error)
}

// ProjectFinder = fitur project.
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

// AlertTypeFinder = fitur alerttype.
type AlertTypeFinder interface {
	Find(ctx context.Context, code string) (*alerttypedomain.AlertType, error)
}

// ZoneChecker = fitur maparea.
type ZoneChecker interface {
	ZoneExists(ctx context.Context, projectID uuid.UUID, name string) (exists, hasMap bool, err error)
}

type Service struct {
	repo       Repository
	projects   ProjectFinder
	alertTypes AlertTypeFinder
	zones      ZoneChecker
}

func New(repo Repository, projects ProjectFinder, alertTypes AlertTypeFinder, zones ZoneChecker) *Service {
	return &Service{repo: repo, projects: projects, alertTypes: alertTypes, zones: zones}
}

// AlertRuleInput = isi LENGKAP aturan (tambah & ganti). Ambang kosong = ikut bawaan / aturan project.
type AlertRuleInput struct {
	ProjectID        *uuid.UUID  `json:"project_id"` // superadmin, saat tambah
	AlertType        string      `json:"alert_type"`
	ZoneName         *string     `json:"zone_name"` // kosong = seluruh project
	DurationMinutes  *float64    `json:"duration_minutes"`
	SpeedLimitKph    *float64    `json:"speed_limit_kph"`
	FuelDropLiters   *float64    `json:"fuel_drop_liters"`
	Description      *string     `json:"description"`
	IsActive         *bool       `json:"is_active"` // kosong = aktif
	RecipientUserIDs []uuid.UUID `json:"recipient_user_ids"`
}

func (s *Service) List(ctx context.Context, a access.Actor, projectID *uuid.UUID) ([]domain.AlertRule, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, pid)
}

// Create menambah aturan. warnings = catatan yang tidak menggagalkan (mis. penerima tanpa nomor HP).
func (s *Service) Create(ctx context.Context, a access.Actor, in AlertRuleInput) (*domain.AlertRule, []string, error) {
	pid, err := access.TargetProject(a, in.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	return s.saveRule(ctx, a, domain.AlertRule{ProjectID: pid}, in)
}

// Replace mengganti seluruh isi aturan (PUT).
func (s *Service) Replace(ctx context.Context, a access.Actor, id uuid.UUID, in AlertRuleInput) (*domain.AlertRule, []string, error) {
	cur, err := s.rule(ctx, a, id)
	if err != nil {
		return nil, nil, err
	}
	return s.saveRule(ctx, a, domain.AlertRule{ID: cur.ID, ProjectID: cur.ProjectID}, in)
}

func (s *Service) Delete(ctx context.Context, a access.Actor, id uuid.UUID) error {
	cur, err := s.rule(ctx, a, id)
	if err != nil {
		return err
	}
	if !a.CanManageProject(cur.ProjectID) {
		return apperror.ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

// Effective = ambang yang berlaku untuk project (+ zona) beserta penerimanya.
func (s *Service) Effective(ctx context.Context, a access.Actor, projectID *uuid.UUID, zoneName string) ([]domain.EffectiveAlertRule, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.Effective(ctx, pid, strings.TrimSpace(zoneName))
}

func (s *Service) rule(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.AlertRule, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || !a.CanView(cur.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	return cur, nil
}

func (s *Service) saveRule(ctx context.Context, a access.Actor, r domain.AlertRule, in AlertRuleInput) (*domain.AlertRule, []string, error) {
	if !a.CanManageProject(r.ProjectID) {
		return nil, nil, apperror.ErrForbidden
	}
	if p, err := s.projects.FindActive(ctx, r.ProjectID); err != nil {
		return nil, nil, err
	} else if p == nil {
		return nil, nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
	}
	t, err := s.alertTypes.Find(ctx, in.AlertType)
	if err != nil {
		return nil, nil, apperror.Invalid("alert_type %q tidak dikenal", in.AlertType)
	}
	r.AlertType = t.Code
	if err := alerttypedomain.ApplyThresholds(t, &r.DurationMinutes, &r.SpeedLimitKph, &r.FuelDropLiters,
		in.DurationMinutes, in.SpeedLimitKph, in.FuelDropLiters, ""); err != nil {
		return nil, nil, err
	}
	var warnings []string
	if z := validate.TrimPtr(in.ZoneName); z != nil && *z != "" {
		if len([]rune(*z)) > 100 {
			return nil, nil, apperror.Invalid("zone_name maks. 100 karakter")
		}
		exists, hasMap, err := s.zones.ZoneExists(ctx, r.ProjectID, *z)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case hasMap && !exists:
			return nil, nil, apperror.Invalid("zona %q tidak ada di peta project yang berlaku", *z)
		case !hasMap:
			warnings = append(warnings, "project belum punya peta; nama zona belum bisa diperiksa")
		}
		r.ZoneName = z
	}
	if d := validate.TrimPtr(in.Description); d != nil && *d != "" {
		if len([]rune(*d)) > 255 {
			return nil, nil, apperror.Invalid("description maks. 255 karakter")
		}
		r.Description = d
	}
	r.IsActive = in.IsActive == nil || *in.IsActive

	// penerima: unik, anggota aktif project
	seen := map[uuid.UUID]bool{}
	for _, id := range in.RecipientUserIDs {
		if !seen[id] {
			seen[id] = true
			r.RecipientUserIDs = append(r.RecipientUserIDs, id)
		}
	}
	if len(r.RecipientUserIDs) > 50 {
		return nil, nil, apperror.Invalid("penerima maks. 50 user")
	}
	if r.RecipientUserIDs == nil {
		r.RecipientUserIDs = []uuid.UUID{}
	}
	members, err := s.repo.ProjectMembers(ctx, r.ProjectID, r.RecipientUserIDs)
	if err != nil {
		return nil, nil, err
	}
	for _, id := range r.RecipientUserIDs {
		m, ok := members[id]
		if !ok {
			return nil, nil, apperror.Invalid("penerima %s bukan anggota aktif project ini", id)
		}
		if m.PhoneNumber == nil || *m.PhoneNumber == "" {
			warnings = append(warnings, m.Name+" belum punya nomor HP; tidak akan menerima WhatsApp")
		}
	}

	id, err := s.repo.Save(ctx, r, a.UserID)
	if err != nil {
		return nil, nil, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, warnings, err
}
