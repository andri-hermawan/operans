package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	authdomain "simha/core-api/internal/auth/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/user/domain"
)

// userMem = database palsu untuk kelola user.
type userMem struct {
	users    map[uuid.UUID]*domain.UserSummary
	active   map[uuid.UUID]bool
	projects map[uuid.UUID]*projectdomain.Project
	revoked  []uuid.UUID
}

func newUserMem() *userMem {
	return &userMem{users: map[uuid.UUID]*domain.UserSummary{}, active: map[uuid.UUID]bool{},
		projects: map[uuid.UUID]*projectdomain.Project{
			projCHR: {ID: projCHR, Code: "CHR", Name: "CHR"},
			projCMN: {ID: projCMN, Code: "CMN", Name: "CMN"},
		}}
}

func (m *userMem) add(email string, superadmin bool, ms ...access.Membership) uuid.UUID {
	id := uuid.New()
	m.users[id] = &domain.UserSummary{ID: id, Email: email, Name: email, IsSuperadmin: superadmin, IsActive: true, Memberships: ms}
	return id
}

func (m *userMem) List(_ context.Context, p *uuid.UUID, _ string) ([]domain.UserSummary, error) {
	out := []domain.UserSummary{}
	for _, u := range m.users {
		if p == nil || len(onlyProject(u.Memberships, *p)) > 0 {
			c := *u
			c.Memberships = append([]access.Membership{}, u.Memberships...)
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *userMem) GetSummary(_ context.Context, id uuid.UUID) (*domain.UserSummary, error) {
	if u, ok := m.users[id]; ok {
		c := *u
		c.Memberships = append([]access.Membership{}, u.Memberships...)
		return &c, nil
	}
	return nil, nil
}
func (m *userMem) FindUserByEmail(_ context.Context, email string) (*authdomain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return &authdomain.User{ID: u.ID, Email: u.Email, IsActive: u.IsActive}, nil
		}
	}
	return nil, nil
}
func (m *userMem) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return m.projects[id], nil
}
func (m *userMem) Insert(_ context.Context, u authdomain.User, _ uuid.UUID, p *uuid.UUID, role string) (uuid.UUID, error) {
	for _, x := range m.users {
		if x.Email == u.Email {
			return uuid.Nil, apperror.ErrEmailExists
		}
	}
	var ms []access.Membership
	if p != nil {
		ms = append(ms, member(*p, m.projects[*p].Code, role))
	}
	return m.add(u.Email, u.IsSuperadmin, ms...), nil
}
func (m *userMem) Update(_ context.Context, id uuid.UUID, upd domain.UserUpdate, _ uuid.UUID) error {
	if upd.IsActive != nil {
		m.users[id].IsActive = *upd.IsActive
	}
	if upd.Name != nil {
		m.users[id].Name = *upd.Name
	}
	return nil
}
func (m *userMem) SetMembership(_ context.Context, uid, pid uuid.UUID, role string, _ *uuid.UUID) error {
	u := m.users[uid]
	for i := range u.Memberships {
		if u.Memberships[i].ProjectID == pid {
			u.Memberships[i].Role = role
			return nil
		}
	}
	u.Memberships = append(u.Memberships, member(pid, m.projects[pid].Code, role))
	return nil
}
func (m *userMem) RemoveMembership(_ context.Context, uid, pid uuid.UUID) (bool, error) {
	u := m.users[uid]
	for i := range u.Memberships {
		if u.Memberships[i].ProjectID == pid {
			u.Memberships = append(u.Memberships[:i], u.Memberships[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}
func (m *userMem) SetPasswordByID(context.Context, uuid.UUID, string) error { return nil }
func (m *userMem) RevokeUserSessions(_ context.Context, id uuid.UUID, _ string) error {
	m.revoked = append(m.revoked, id)
	return nil
}

func TestKelolaUser(t *testing.T) {
	ctx := context.Background()
	st := newUserMem()
	s := New(st, st, st)

	saID := st.add("sa@simha.id", true)
	adminCHR := st.add("admin.chr@simha.id", false, member(projCHR, "CHR", access.RoleAdmin))
	dispCHR := st.add("disp.chr@simha.id", false, member(projCHR, "CHR", access.RoleDispatcher), member(projCMN, "CMN", access.RoleViewer))
	viewCMN := st.add("view.cmn@simha.id", false, member(projCMN, "CMN", access.RoleViewer))

	sa := access.Actor{UserID: saID, SuperAdmin: true}
	admin := access.Actor{UserID: adminCHR, ProjectID: &projCHR, Role: access.RoleAdmin}
	dispatcher := access.Actor{UserID: dispCHR, ProjectID: &projCHR, Role: access.RoleDispatcher}

	t.Run("dispatcher/viewer tidak boleh membuka kelola user", func(t *testing.T) {
		if _, err := s.List(ctx, dispatcher, nil, ""); err != apperror.ErrForbidden {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("admin CHR hanya melihat anggota CHR, dan hanya keanggotaan CHR", func(t *testing.T) {
		us, err := s.List(ctx, admin, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(us) != 2 {
			t.Fatalf("seharusnya 2 anggota CHR, dapat %d", len(us))
		}
		for _, u := range us {
			if len(u.Memberships) != 1 || u.Memberships[0].ProjectCode != "CHR" {
				t.Fatalf("keanggotaan project lain bocor: %+v", u.Memberships)
			}
		}
		if _, err := s.Get(ctx, admin, viewCMN); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("user project lain seharusnya 'tidak ditemukan', err = %v", err)
		}
	})

	t.Run("superadmin melihat semua user", func(t *testing.T) {
		if us, _ := s.List(ctx, sa, nil, ""); len(us) != 4 {
			t.Fatalf("dapat %d", len(us))
		}
	})

	t.Run("admin membuat user → otomatis masuk project CHR; tidak bisa membuat superadmin", func(t *testing.T) {
		u, err := s.Create(ctx, admin, NewUser{Email: "Baru@SIMHA.id", Name: "Baru", Password: "password-baru", ProjectID: &projCMN, Role: access.RoleViewer})
		if err != nil {
			t.Fatal(err)
		}
		if u.Email != "baru@simha.id" || len(u.Memberships) != 1 || u.Memberships[0].ProjectID != projCHR {
			t.Fatalf("hasil: %+v", u)
		}
		if _, err := s.Create(ctx, admin, NewUser{Email: "x@simha.id", Name: "X", Password: "password-baru", IsSuperadmin: true, Role: access.RoleViewer}); err != apperror.ErrForbidden {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("isian tidak valid & email ganda ditolak", func(t *testing.T) {
		var ve *apperror.ValidationError
		cases := []NewUser{
			{Email: "bukan-email", Name: "A", Password: "password-baru", Role: access.RoleViewer},
			{Email: "a1@simha.id", Name: "A", Password: "pendek", Role: access.RoleViewer},
			{Email: "a2@simha.id", Name: "A", Password: "password-baru", Role: "bos"},
			{Email: "a3@simha.id", Name: "A", Password: "password-baru", Role: access.RoleViewer, PhoneNumber: ptr("08-abc")},
		}
		for _, c := range cases {
			if _, err := s.Create(ctx, admin, c); !errors.As(err, &ve) {
				t.Errorf("%+v → seharusnya ValidationError, dapat %v", c, err)
			}
		}
		if _, err := s.Create(ctx, admin, NewUser{Email: "disp.chr@simha.id", Name: "A", Password: "password-baru", Role: access.RoleViewer}); !errors.Is(err, apperror.ErrEmailExists) {
			t.Fatalf("email ganda: err = %v", err)
		}
	})

	t.Run("hanya superadmin yang boleh menonaktifkan; tidak untuk diri sendiri; sesi dicabut", func(t *testing.T) {
		off := false
		if _, err := s.Update(ctx, admin, dispCHR, domain.UserUpdate{IsActive: &off}); err != apperror.ErrForbidden {
			t.Fatalf("admin: err = %v", err)
		}
		if _, err := s.Update(ctx, sa, saID, domain.UserUpdate{IsActive: &off}); err != apperror.ErrSelf {
			t.Fatalf("diri sendiri: err = %v", err)
		}
		if _, err := s.Update(ctx, sa, dispCHR, domain.UserUpdate{IsActive: &off}); err != nil {
			t.Fatal(err)
		}
		if len(st.revoked) != 1 || st.revoked[0] != dispCHR {
			t.Fatal("user yang dinonaktifkan harus langsung keluar dari semua sesi")
		}
		on := true
		s.Update(ctx, sa, dispCHR, domain.UserUpdate{IsActive: &on})
	})

	t.Run("admin hanya mengatur anggota project-nya; tidak boleh mengubah role sendiri", func(t *testing.T) {
		if _, err := s.SetMember(ctx, admin, projCMN, "view.cmn@simha.id", access.RoleAdmin); err != apperror.ErrForbidden {
			t.Fatalf("project lain: err = %v", err)
		}
		if _, err := s.SetMember(ctx, admin, projCHR, "admin.chr@simha.id", access.RoleViewer); err != apperror.ErrSelf {
			t.Fatalf("diri sendiri: err = %v", err)
		}
		u, err := s.SetMember(ctx, admin, projCHR, "view.cmn@simha.id", access.RoleDispatcher)
		if err != nil || len(u.Memberships) != 1 || u.Memberships[0].Role != access.RoleDispatcher {
			t.Fatalf("tambah anggota: %+v err=%v", u, err)
		}
		if err := s.RemoveMember(ctx, admin, projCHR, viewCMN); err != nil {
			t.Fatal(err)
		}
		if err := s.RemoveMember(ctx, admin, projCHR, adminCHR); err != apperror.ErrSelf {
			t.Fatalf("keluarkan diri sendiri: err = %v", err)
		}
	})

	t.Run("reset password hanya oleh superadmin", func(t *testing.T) {
		if err := s.ResetPassword(ctx, admin, dispCHR, "password-baru"); err != apperror.ErrForbidden {
			t.Fatalf("err = %v", err)
		}
		if err := s.ResetPassword(ctx, sa, dispCHR, "password-baru"); err != nil {
			t.Fatal(err)
		}
	})
}

func ptr[T any](v T) *T { return &v }

var (
	projCHR = uuid.New()
	projCMN = uuid.New()
)

func member(p uuid.UUID, code, role string) access.Membership {
	return access.Membership{ProjectID: p, ProjectCode: code, ProjectName: code, Role: role}
}
