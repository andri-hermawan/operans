package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"simha/core-api/internal/pagerule/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

type fakeRepo struct {
	roleRules, userRules []domain.Rule
	member               bool
	applied              *domain.SaveInput
}

func (f *fakeRepo) ListProject(context.Context, uuid.UUID) ([]domain.Rule, error) {
	return append(append([]domain.Rule{}, f.roleRules...), f.userRules...), nil
}
func (f *fakeRepo) ListFor(context.Context, uuid.UUID, string, uuid.UUID) ([]domain.Rule, []domain.Rule, error) {
	return f.roleRules, f.userRules, nil
}
func (f *fakeRepo) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.member, nil
}
func (f *fakeRepo) Apply(_ context.Context, _ uuid.UUID, in domain.SaveInput, _ uuid.UUID) error {
	f.applied = &in
	return nil
}

func boolp(b bool) *bool { return &b }

func superadmin() access.Actor { return access.Actor{UserID: uuid.New(), SuperAdmin: true} }

func TestMine(t *testing.T) {
	pid := uuid.New()
	repo := &fakeRepo{
		roleRules: []domain.Rule{{PageKey: "report/report-ritase-daily", Allowed: false}},
		userRules: []domain.Rule{{PageKey: "report/report-ritase-daily", Allowed: true}},
	}
	s := New(repo)

	got, err := s.Mine(context.Background(), access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: "viewer"})
	if err != nil || !got["report/report-ritase-daily"] {
		t.Fatalf("aturan user harus menimpa role: %v %v", got, err)
	}
	// superadmin selalu penuh → tidak ada perubahan
	if got, _ := s.Mine(context.Background(), superadmin()); len(got) != 0 {
		t.Fatalf("superadmin tidak boleh dibatasi: %v", got)
	}
	// tanpa project aktif
	if got, _ := s.Mine(context.Background(), access.Actor{UserID: uuid.New()}); len(got) != 0 {
		t.Fatalf("tanpa project = kosong: %v", got)
	}
}

func TestListDanSave_HanyaSuperadmin(t *testing.T) {
	pid := uuid.New()
	s := New(&fakeRepo{})
	admin := access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: "admin"}
	if _, err := s.List(context.Background(), admin, &pid); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("admin tidak boleh melihat aturan: %v", err)
	}
	in := domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer", Changes: []domain.Change{{PageKey: "tracking", Allowed: boolp(false)}}}
	if err := s.Save(context.Background(), admin, in); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("admin tidak boleh menyimpan: %v", err)
	}
}

func TestSave_Validasi(t *testing.T) {
	pid := uuid.New()
	uid := uuid.New()
	ch := func(k string) []domain.Change { return []domain.Change{{PageKey: k, Allowed: boolp(true)}} }
	for name, c := range map[string]struct {
		in     domain.SaveInput
		member bool
		ok     bool
	}{
		"role valid":          {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer", Changes: ch("tracking")}, false, true},
		"role tidak dikenal":  {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "superadmin", Changes: ch("tracking")}, false, false},
		"user anggota":        {domain.SaveInput{ProjectID: &pid, SubjectType: "user", UserID: &uid, Changes: ch("tracking")}, true, true},
		"user bukan anggota":  {domain.SaveInput{ProjectID: &pid, SubjectType: "user", UserID: &uid, Changes: ch("tracking")}, false, false},
		"user tanpa id":       {domain.SaveInput{ProjectID: &pid, SubjectType: "user", Changes: ch("tracking")}, true, false},
		"jenis salah":         {domain.SaveInput{ProjectID: &pid, SubjectType: "grup", Role: "viewer", Changes: ch("tracking")}, false, false},
		"kunci tidak valid":   {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer", Changes: ch("Track ing")}, false, false},
		"halaman terlindung":  {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "admin", Changes: ch("master/access")}, false, false},
		"tanpa perubahan":     {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer"}, false, false},
		"kunci dobel":         {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer", Changes: append(ch("tracking"), ch("tracking")...)}, false, false},
		"hapus (allowed nil)": {domain.SaveInput{ProjectID: &pid, SubjectType: "role", Role: "viewer", Changes: []domain.Change{{PageKey: "tracking"}}}, false, true},
	} {
		repo := &fakeRepo{member: c.member}
		err := New(repo).Save(context.Background(), superadmin(), c.in)
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v, mau ok=%v", name, err, c.ok)
		}
		if c.ok && repo.applied == nil {
			t.Errorf("%s: seharusnya tersimpan", name)
		}
		if !c.ok && repo.applied != nil {
			t.Errorf("%s: tidak boleh tersimpan", name)
		}
	}
}
