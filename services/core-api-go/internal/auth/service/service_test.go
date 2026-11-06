package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"simha/core-api/internal/auth/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/token"
	"simha/core-api/pkg/hash"
)

// memStore = database palsu di memori.
type memStore struct {
	mu       sync.Mutex
	users    map[uuid.UUID]*domain.User
	members  map[uuid.UUID][]access.Membership
	projects map[uuid.UUID]*projectdomain.Project
	tokens   map[string]*domain.RefreshToken // kunci: hash
}

func newMemStore() *memStore {
	return &memStore{users: map[uuid.UUID]*domain.User{}, members: map[uuid.UUID][]access.Membership{},
		projects: map[uuid.UUID]*projectdomain.Project{}, tokens: map[string]*domain.RefreshToken{}}
}

func (m *memStore) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			c := *u
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memStore) GetUser(_ context.Context, id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[id]; ok {
		c := *u
		return &c, nil
	}
	return nil, nil
}
func (m *memStore) SetLoginFailures(_ context.Context, id uuid.UUID, count int, lock *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id].FailedLoginCount, m.users[id].LockedUntil = count, lock
	return nil
}
func (m *memStore) RecordLoginSuccess(_ context.Context, id uuid.UUID, newHash *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.users[id]
	u.FailedLoginCount, u.LockedUntil = 0, nil
	if newHash != nil {
		u.PasswordHash = *newHash
	}
	return nil
}
func (m *memStore) ListMemberships(_ context.Context, id uuid.UUID) ([]access.Membership, error) {
	return append([]access.Membership{}, m.members[id]...), nil
}
func (m *memStore) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return m.projects[id], nil
}
func (m *memStore) InsertRefreshToken(_ context.Context, t domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[t.TokenHash] = &t
	return nil
}
func (m *memStore) FindRefreshToken(_ context.Context, hash string) (*domain.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tokens[hash]; ok {
		c := *t
		return &c, nil
	}
	return nil, nil
}
func (m *memStore) RotateRefreshToken(_ context.Context, oldID uuid.UUID, next domain.RefreshToken) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tokens {
		if t.ID == oldID {
			if t.UsedAt != nil || t.RevokedAt != nil {
				return false, nil
			}
			now := time.Now()
			t.UsedAt = &now
			m.tokens[next.TokenHash] = &next
			return true, nil
		}
	}
	return false, nil
}
func (m *memStore) SetPasswordByID(_ context.Context, id uuid.UUID, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id].PasswordHash = hash
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == id && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *memStore) RevokeFamily(_ context.Context, family uuid.UUID, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, t := range m.tokens {
		if t.FamilyID == family && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

var (
	projCHR = uuid.New()
	projCMN = uuid.New()
	projX   = uuid.New() // project yang user tidak menjadi anggota
)

func setup(t *testing.T) (*Service, *memStore, *token.Signer) {
	t.Helper()
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := token.NewSigner(key, "simha-core", "simha-api", 15*time.Minute)
	st := newMemStore()
	for id, code := range map[uuid.UUID]string{projCHR: "CHR", projCMN: "CMN", projX: "XXX"} {
		st.projects[id] = &projectdomain.Project{ID: id, Code: code, Name: code}
	}
	s := New(st, st, signer, Policy{RefreshIdleTTL: 7 * 24 * time.Hour, RefreshMaxTTL: 30 * 24 * time.Hour,
		MaxFailedLogins: 3, LockDuration: 15 * time.Minute})
	return s, st, signer
}

func addUser(st *memStore, email, pw string, superadmin bool, ms ...access.Membership) *domain.User {
	h, _ := hash.HashPassword(pw)
	u := &domain.User{ID: uuid.New(), Email: email, Name: email, PasswordHash: h, IsSuperadmin: superadmin, IsActive: true}
	st.users[u.ID] = u
	st.members[u.ID] = ms
	return u
}

func member(p uuid.UUID, code, role string) access.Membership {
	return access.Membership{ProjectID: p, ProjectCode: code, ProjectName: code, Role: role}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("berhasil: token berisi project & role", func(t *testing.T) {
		s, st, signer := setup(t)
		addUser(st, "dispatcher@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleDispatcher))
		sess, err := s.Login(ctx, "  Dispatcher@SIMHA.id ", "password-benar", nil, Client{})
		if err != nil {
			t.Fatal(err)
		}
		c, _ := signer.Verify(sess.AccessToken)
		if c.ProjectID != projCHR.String() || c.ProjectRole != access.RoleDispatcher || c.SuperAdmin || sess.RefreshToken == "" {
			t.Fatalf("token salah: %+v", c)
		}
	})

	t.Run("password salah & email tidak ada → pesan yang SAMA", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "a@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleViewer))
		_, e1 := s.Login(ctx, "a@simha.id", "salah", nil, Client{})
		_, e2 := s.Login(ctx, "tidak-ada@simha.id", "salah", nil, Client{})
		if e1 != ErrInvalidCredentials || e2 != ErrInvalidCredentials {
			t.Fatalf("e1=%v e2=%v", e1, e2)
		}
	})

	t.Run("3x gagal → dikunci, password benar pun ditolak", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "b@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleViewer))
		for i := 0; i < 3; i++ {
			s.Login(ctx, "b@simha.id", "salah", nil, Client{})
		}
		var locked *LockedError
		if _, err := s.Login(ctx, "b@simha.id", "password-benar", nil, Client{}); !errors.As(err, &locked) {
			t.Fatalf("seharusnya terkunci, err = %v", err)
		}
	})

	t.Run("password bcrypt sistem lama diterima lalu diganti argon2id", func(t *testing.T) {
		s, st, _ := setup(t)
		u := addUser(st, "lama@simha.id", "x", false, member(projCHR, "CHR", access.RoleAdmin))
		old, _ := bcrypt.GenerateFromPassword([]byte("password-lama"), bcrypt.MinCost)
		u.PasswordHash = string(old)
		if _, err := s.Login(ctx, "lama@simha.id", "password-lama", nil, Client{}); err != nil {
			t.Fatal(err)
		}
		if h := st.users[u.ID].PasswordHash; h[:9] != "$argon2id" {
			t.Fatalf("hash seharusnya sudah argon2id, sekarang %.9s", h)
		}
	})

	t.Run("user nonaktif ditolak", func(t *testing.T) {
		s, st, _ := setup(t)
		u := addUser(st, "off@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleViewer))
		u.IsActive = false
		if _, err := s.Login(ctx, "off@simha.id", "password-benar", nil, Client{}); err != ErrInvalidCredentials {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("bukan superadmin & tanpa project → ditolak", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "kosong@simha.id", "password-benar", false)
		if _, err := s.Login(ctx, "kosong@simha.id", "password-benar", nil, Client{}); err != ErrNoProject {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("superadmin tanpa project → token lintas project", func(t *testing.T) {
		s, st, signer := setup(t)
		addUser(st, "sa@simha.id", "password-benar", true)
		sess, err := s.Login(ctx, "sa@simha.id", "password-benar", nil, Client{})
		if err != nil {
			t.Fatal(err)
		}
		if c, _ := signer.Verify(sess.AccessToken); !c.SuperAdmin || c.ProjectID != "" || sess.Project != nil {
			t.Fatalf("token superadmin salah: %+v", c)
		}
	})
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()

	t.Run("rotation: token lama tidak berlaku lagi; pemakaian ulang mencabut SELURUH sesi", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "d@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleDispatcher))
		s1, _ := s.Login(ctx, "d@simha.id", "password-benar", nil, Client{})

		s2, err := s.Refresh(ctx, s1.RefreshToken, nil, Client{})
		if err != nil || s2.RefreshToken == s1.RefreshToken {
			t.Fatalf("refresh pertama: err=%v", err)
		}
		// "Pencuri" memakai token lama → ditolak & seluruh keluarga dicabut
		if _, err := s.Refresh(ctx, s1.RefreshToken, nil, Client{}); err != ErrRefreshReused {
			t.Fatalf("pemakaian ulang seharusnya ErrRefreshReused, dapat %v", err)
		}
		// Token terbaru milik pemilik sah pun ikut tidak berlaku → harus login ulang
		if _, err := s.Refresh(ctx, s2.RefreshToken, nil, Client{}); err != ErrInvalidRefresh {
			t.Fatalf("token terbaru seharusnya ikut dicabut, dapat %v", err)
		}
	})

	t.Run("switch project: anggota boleh, bukan anggota ditolak", func(t *testing.T) {
		s, st, signer := setup(t)
		addUser(st, "e@simha.id", "password-benar", false,
			member(projCHR, "CHR", access.RoleDispatcher), member(projCMN, "CMN", access.RoleViewer))
		s1, _ := s.Login(ctx, "e@simha.id", "password-benar", nil, Client{})
		s2, err := s.Refresh(ctx, s1.RefreshToken, &projCMN, Client{})
		if err != nil {
			t.Fatal(err)
		}
		if c, _ := signer.Verify(s2.AccessToken); c.ProjectID != projCMN.String() || c.ProjectRole != access.RoleViewer {
			t.Fatalf("setelah pindah project: %+v", c)
		}
		if _, err := s.Refresh(ctx, s2.RefreshToken, &projX, Client{}); err != ErrProjectForbidden {
			t.Fatalf("project bukan anggota seharusnya ditolak, dapat %v", err)
		}
	})

	t.Run("superadmin boleh membuka project mana pun", func(t *testing.T) {
		s, st, signer := setup(t)
		addUser(st, "sa2@simha.id", "password-benar", true)
		s1, _ := s.Login(ctx, "sa2@simha.id", "password-benar", nil, Client{})
		s2, err := s.Refresh(ctx, s1.RefreshToken, &projX, Client{})
		if err != nil {
			t.Fatal(err)
		}
		if c, _ := signer.Verify(s2.AccessToken); c.ProjectID != projX.String() || !c.SuperAdmin {
			t.Fatalf("token: %+v", c)
		}
	})

	t.Run("logout mencabut sesi", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "f@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleViewer))
		s1, _ := s.Login(ctx, "f@simha.id", "password-benar", nil, Client{})
		if err := s.Logout(ctx, s1.RefreshToken); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Refresh(ctx, s1.RefreshToken, nil, Client{}); err != ErrInvalidRefresh {
			t.Fatalf("setelah logout seharusnya ditolak, dapat %v", err)
		}
	})

	t.Run("refresh token kedaluwarsa ditolak", func(t *testing.T) {
		s, st, _ := setup(t)
		addUser(st, "g@simha.id", "password-benar", false, member(projCHR, "CHR", access.RoleViewer))
		s1, _ := s.Login(ctx, "g@simha.id", "password-benar", nil, Client{})
		s.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) } // 8 hari tidak dipakai
		if _, err := s.Refresh(ctx, s1.RefreshToken, nil, Client{}); err != ErrInvalidRefresh {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("ganti password sendiri: password lama salah ditolak; berhasil → semua sesi berakhir", func(t *testing.T) {
		s, st, _ := setup(t)
		u := addUser(st, "h@simha.id", "password-lama", false, member(projCHR, "CHR", access.RoleViewer))
		s1, _ := s.Login(ctx, "h@simha.id", "password-lama", nil, Client{})
		if err := s.ChangePassword(ctx, u.ID, "tebakan", "password-baru-1"); err != ErrWrongPassword {
			t.Fatalf("err = %v", err)
		}
		var ve *apperror.ValidationError
		if err := s.ChangePassword(ctx, u.ID, "password-lama", "pendek"); !errors.As(err, &ve) {
			t.Fatalf("password pendek seharusnya ditolak, err = %v", err)
		}
		if err := s.ChangePassword(ctx, u.ID, "password-lama", "password-baru-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Refresh(ctx, s1.RefreshToken, nil, Client{}); err != ErrInvalidRefresh {
			t.Fatalf("sesi lama seharusnya berakhir, err = %v", err)
		}
		if _, err := s.Login(ctx, "h@simha.id", "password-baru-1", nil, Client{}); err != nil {
			t.Fatalf("login dengan password baru: %v", err)
		}
	})

	t.Run("token palsu ditolak", func(t *testing.T) {
		s, _, _ := setup(t)
		if _, err := s.Refresh(ctx, "token-karangan", nil, Client{}); err != ErrInvalidRefresh {
			t.Fatalf("err = %v", err)
		}
	})
}

func (m *memStore) RevokeUserSessions(_ context.Context, id uuid.UUID, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == id && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}
