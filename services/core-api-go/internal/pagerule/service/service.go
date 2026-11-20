// Package service (fitur pagerule) = hak akses halaman & laporan per project: user membaca hak miliknya sendiri,
// hanya superadmin yang boleh melihat seluruh aturan dan mengubahnya. Aturan dipakai frontend (menu & halaman) dan
// penjaga di API (Allowed + delivery/http/guard.go) untuk data Report dan Record / Upload.
package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/pagerule/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

const (
	maxChanges = 200
	cacheTTL   = 30 * time.Second // aturan dibaca dari database paling sering sekali per 30 detik per user
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	ListProject(ctx context.Context, projectID uuid.UUID) ([]domain.Rule, error)
	ListFor(ctx context.Context, projectID uuid.UUID, role string, userID uuid.UUID) (roleRules, userRules []domain.Rule, err error)
	IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	Apply(ctx context.Context, projectID uuid.UUID, in domain.SaveInput, by uuid.UUID) error
}

type Service struct {
	repo Repository
	now  func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	role, user map[string]bool
	at         time.Time
}

func New(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now, cache: map[string]cacheEntry{}}
}

func ruleMap(rules []domain.Rule) map[string]bool {
	m := make(map[string]bool, len(rules))
	for _, r := range rules {
		m[r.PageKey] = r.Allowed
	}
	return m
}

// rulesOf = aturan role + aturan user pemanggil di project aktifnya (di-cache sebentar).
func (s *Service) rulesOf(ctx context.Context, a access.Actor) (role, user map[string]bool, err error) {
	ck := a.ProjectID.String() + "|" + a.Role + "|" + a.UserID.String()
	s.mu.Lock()
	e, ok := s.cache[ck]
	s.mu.Unlock()
	if ok && s.now().Sub(e.at) < cacheTTL {
		return e.role, e.user, nil
	}
	rr, ur, err := s.repo.ListFor(ctx, *a.ProjectID, a.Role, a.UserID)
	if err != nil {
		return nil, nil, err
	}
	e = cacheEntry{role: ruleMap(rr), user: ruleMap(ur), at: s.now()}
	s.mu.Lock()
	s.cache[ck] = e
	s.mu.Unlock()
	return e.role, e.user, nil
}

func (s *Service) clearCache() {
	s.mu.Lock()
	s.cache = map[string]cacheEntry{}
	s.mu.Unlock()
}

// Allowed = boleh tidaknya pemanggil memakai data halaman-halaman ini: SALAH SATU halaman terbuka sudah cukup.
// Superadmin dan token layanan (cetak PDF terjadwal) selalu boleh; tanpa project aktif selalu ditolak.
func (s *Service) Allowed(ctx context.Context, a access.Actor, keys []string) (bool, error) {
	if a.SuperAdmin || a.Internal {
		return true, nil
	}
	if a.ProjectID == nil {
		return false, nil
	}
	role, user, err := s.rulesOf(ctx, a)
	if err != nil {
		return false, err
	}
	for _, k := range keys {
		if domain.Resolve(a.Role, k, role, user) {
			return true, nil
		}
	}
	return false, nil
}

// Mine = perubahan dari aturan bawaan yang berlaku untuk pemanggil di project aktifnya (kunci → boleh/tidak).
// Superadmin selalu penuh, jadi hasilnya kosong.
func (s *Service) Mine(ctx context.Context, a access.Actor) (map[string]bool, error) {
	if a.SuperAdmin || a.ProjectID == nil {
		return map[string]bool{}, nil
	}
	roleRules, userRules, err := s.repo.ListFor(ctx, *a.ProjectID, a.Role, a.UserID)
	if err != nil {
		return nil, err
	}
	return domain.Effective(roleRules, userRules), nil
}

// List = seluruh aturan satu project (khusus superadmin).
func (s *Service) List(ctx context.Context, a access.Actor, projectID *uuid.UUID) ([]domain.Rule, error) {
	if !a.SuperAdmin {
		return nil, apperror.ErrForbidden
	}
	pid, err := access.TargetProject(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListProject(ctx, pid)
}

// Save menyimpan perubahan untuk satu role atau satu user di satu project (khusus superadmin).
func (s *Service) Save(ctx context.Context, a access.Actor, in domain.SaveInput) error {
	if !a.SuperAdmin {
		return apperror.ErrForbidden
	}
	pid, err := access.TargetProject(a, in.ProjectID)
	if err != nil {
		return err
	}
	switch in.SubjectType {
	case domain.SubjectRole:
		if !domain.ValidRole(in.Role) || in.UserID != nil {
			return apperror.Invalid("role tidak valid (admin, dispatcher, viewer)")
		}
	case domain.SubjectUser:
		if in.UserID == nil || in.Role != "" {
			return apperror.Invalid("user_id wajib diisi untuk aturan per user")
		}
		ok, err := s.repo.IsMember(ctx, pid, *in.UserID)
		if err != nil {
			return err
		}
		if !ok {
			return apperror.Invalid("user bukan anggota project ini")
		}
	default:
		return apperror.Invalid("subject_type harus 'role' atau 'user'")
	}
	if len(in.Changes) == 0 {
		return apperror.Invalid("tidak ada perubahan")
	}
	if len(in.Changes) > maxChanges {
		return apperror.Invalid("terlalu banyak perubahan sekaligus (maks. %d)", maxChanges)
	}
	seen := make(map[string]bool, len(in.Changes))
	for _, c := range in.Changes {
		switch {
		case !domain.ValidKey(c.PageKey):
			return apperror.Invalid("kunci halaman tidak valid: %q", c.PageKey)
		case domain.ProtectedKeys[c.PageKey]:
			return apperror.Invalid("halaman %q tidak boleh dikunci lewat pengaturan ini", c.PageKey)
		case seen[c.PageKey]:
			return apperror.Invalid("kunci halaman dobel: %q", c.PageKey)
		}
		seen[c.PageKey] = true
	}
	if err := s.repo.Apply(ctx, pid, in, a.UserID); err != nil {
		return err
	}
	s.clearCache() // aturan baru langsung dipakai penjaga di instance ini
	return nil
}
