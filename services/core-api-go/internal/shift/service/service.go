// Package service (fitur shift) = jam kerja project: pilihan shift untuk form (semua role project),
// kelola shift (admin project & superadmin), dan daftar shift aktif untuk fitur lain.
// Shift aktif satu project tidak boleh bertumpuk jamnya.
package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
	"simha/core-api/internal/shift/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	ListActive(ctx context.Context, projectID uuid.UUID) ([]domain.Shift, error)
	ListAll(ctx context.Context, projectID uuid.UUID) ([]domain.ShiftRow, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.ShiftRow, error)
	Insert(ctx context.Context, s domain.ShiftRow, by uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, s domain.ShiftRow, by uuid.UUID) error
}

// ProjectFinder = fitur project.
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

type Service struct {
	repo     Repository
	projects ProjectFinder
}

func New(repo Repository, projects ProjectFinder) *Service {
	return &Service{repo: repo, projects: projects}
}

// ListActive = shift aktif project, urut sequence (dipakai fitur input harian & jadwal laporan).
func (s *Service) ListActive(ctx context.Context, projectID uuid.UUID) ([]domain.Shift, error) {
	return s.repo.ListActive(ctx, projectID)
}

// Options = daftar shift project (untuk pilihan di form). Boleh untuk semua role project.
// Pemeriksaan sama dengan input harian: project harus aktif dan zona waktunya valid.
func (s *Service) Options(ctx context.Context, a access.Actor, projectID *uuid.UUID) ([]domain.Shift, error) {
	pid, err := access.TargetProject(a, projectID)
	if err != nil {
		return nil, err
	}
	if !a.CanView(pid) {
		return nil, apperror.ErrForbidden
	}
	p, err := s.projects.FindActive(ctx, pid)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil {
		return nil, err
	}
	return s.repo.ListActive(ctx, p.ID)
}

// ListAll = semua shift project termasuk yang nonaktif (halaman kelola shift).
func (s *Service) ListAll(ctx context.Context, a access.Actor, projectFilter *uuid.UUID) ([]domain.ShiftRow, error) {
	pid := projectFilter
	if pid == nil {
		pid = a.ProjectID
	}
	if pid == nil {
		return nil, apperror.Invalid("project_id wajib diisi")
	}
	if !a.CanView(*pid) {
		return nil, apperror.ErrForbidden
	}
	return s.repo.ListAll(ctx, *pid)
}

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// Save menambah (id nil) atau mengubah shift. Shift aktif satu project tidak boleh bertumpuk.
func (s *Service) Save(ctx context.Context, a access.Actor, id *uuid.UUID, in domain.ShiftInput) (*domain.ShiftRow, error) {
	var sh domain.ShiftRow
	if id == nil {
		pid := in.ProjectID
		if !a.SuperAdmin || pid == nil {
			pid = a.ProjectID
		}
		if pid == nil {
			return nil, apperror.Invalid("project_id wajib diisi")
		}
		sh = domain.ShiftRow{Shift: domain.Shift{ProjectID: *pid}, IsActive: true}
	} else {
		cur, err := s.repo.Get(ctx, *id)
		if err != nil {
			return nil, err
		}
		if cur == nil || !a.CanView(cur.ProjectID) {
			return nil, apperror.ErrNotFound
		}
		sh = *cur
	}
	if !a.CanManageProject(sh.ProjectID) {
		return nil, apperror.ErrForbidden
	}

	code, err := validate.Code(in.Code, "shift_code")
	if err != nil {
		return nil, err
	}
	name, err := validate.Name(in.Name, "shift_name", 100)
	if err != nil {
		return nil, err
	}
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	set(&sh.Code, code)
	set(&sh.Name, name)
	set(&sh.Start, in.Start)
	set(&sh.End, in.End)
	if in.Sequence != nil {
		sh.Sequence = *in.Sequence
	}
	if in.IsActive != nil {
		sh.IsActive = *in.IsActive
	}
	if sh.Code == "" || sh.Name == "" {
		return nil, apperror.Invalid("shift_code dan shift_name wajib diisi")
	}
	if !hhmm.MatchString(sh.Start) || !hhmm.MatchString(sh.End) || sh.Start == sh.End {
		return nil, apperror.Invalid("start_time & end_time format HH:MM dan tidak boleh sama")
	}
	if sh.Sequence < 1 || sh.Sequence > 9 {
		return nil, apperror.Invalid("sequence 1–9")
	}

	others, err := s.repo.ListAll(ctx, sh.ProjectID)
	if err != nil {
		return nil, err
	}
	if sh.IsActive {
		for _, o := range others {
			if o.ID != sh.ID && o.IsActive && ShiftsOverlap(sh.Shift, o.Shift) {
				return nil, apperror.Invalid("jam shift bertumpuk dengan %s (%s–%s)", o.Name, o.Start, o.End)
			}
		}
	}

	if id == nil {
		newID, err := s.repo.Insert(ctx, sh, a.UserID)
		if err != nil {
			return nil, err
		}
		id = &newID
	} else if err := s.repo.Update(ctx, sh, a.UserID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, *id)
}

// ShiftsOverlap: apakah dua shift (boleh melewati tengah malam) memakai menit yang sama.
func ShiftsOverlap(a, b domain.Shift) bool {
	ma, mb := shiftMinutes(a), shiftMinutes(b)
	for m := range ma {
		if mb[m] {
			return true
		}
	}
	return false
}

func shiftMinutes(s domain.Shift) map[int]bool {
	var sh, sm, eh, em int
	fmt.Sscanf(s.Start, "%d:%d", &sh, &sm)
	fmt.Sscanf(s.End, "%d:%d", &eh, &em)
	start, end := sh*60+sm, eh*60+em
	out := map[int]bool{}
	for m := start; m != end; m = (m + 1) % 1440 {
		out[m] = true
	}
	return out
}
