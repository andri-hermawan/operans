package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/reportschedule/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/token"
	shiftdomain "simha/core-api/internal/shift/domain"
)

type fakeReportStore struct {
	sc   *domain.ReportSchedule
	logs []domain.ReportScheduleLog
	ran  map[string]bool
}

func (f *fakeReportStore) ListActive(context.Context, uuid.UUID) ([]shiftdomain.Shift, error) {
	return []shiftdomain.Shift{{Code: "SHIFT_1", Name: "Shift 1", Sequence: 1}}, nil
}
func (f *fakeReportStore) List(context.Context, *uuid.UUID, bool) ([]domain.ReportSchedule, error) {
	return []domain.ReportSchedule{*f.sc}, nil
}
func (f *fakeReportStore) Get(context.Context, uuid.UUID) (*domain.ReportSchedule, error) {
	return f.sc, nil
}
func (f *fakeReportStore) Insert(_ context.Context, s domain.ReportSchedule, _ uuid.UUID) (uuid.UUID, error) {
	s.ID = uuid.New()
	f.sc = &s
	return s.ID, nil
}
func (f *fakeReportStore) Update(_ context.Context, s domain.ReportSchedule, _ uuid.UUID) error {
	f.sc = &s
	return nil
}
func (f *fakeReportStore) Delete(context.Context, uuid.UUID) error { return nil }
func (f *fakeReportStore) HasScheduledRun(_ context.Context, _ uuid.UUID, d string) (bool, error) {
	return f.ran[d], nil
}
func (f *fakeReportStore) InsertLog(_ context.Context, _ uuid.UUID, l domain.ReportScheduleLog) error {
	f.logs = append(f.logs, l)
	if l.Trigger == "schedule" {
		f.ran[l.RunDate] = true
	}
	return nil
}
func (f *fakeReportStore) ListLogs(context.Context, uuid.UUID) ([]domain.ReportScheduleLog, error) {
	return f.logs, nil
}

type fakeIssuer struct{}

func (fakeIssuer) IssueAccess(token.AccessSubject) (string, time.Time, error) {
	return "tok", time.Now(), nil
}

func newTestReports(smtp bool) (*Service, *fakeReportStore, *[]string) {
	owner := uuid.New()
	shift := "SHIFT_1"
	st := &fakeReportStore{ran: map[string]bool{}, sc: &domain.ReportSchedule{
		ID: uuid.New(), ProjectID: uuid.New(), Name: "Harian", ReportType: "alert_summary", ShiftCode: &shift,
		DateOffset: -1, SendTime: "09:00", IsActive: true, Timezone: "Asia/Jakarta", CreatedBy: &owner,
		Recipients: []domain.ReportRecipient{{Email: "a@x.id"}},
	}}
	cfg := Config{FrontendURL: "http://web"}
	if smtp {
		cfg.SMTP.Host = "smtp.x"
	}
	r := New(st, st, fakeIssuer{}, cfg)
	var urls []string
	r.render = func(_ context.Context, u string) ([]byte, error) { urls = append(urls, u); return []byte("%PDF"), nil }
	r.send = func([]string, string, string, string, []byte) error { return nil }
	return r, st, &urls
}

func TestReportSchedulerTick(t *testing.T) {
	r, st, urls := newTestReports(true)
	wib := time.FixedZone("WIB", 7*3600)

	r.now = func() time.Time { return time.Date(2026, 10, 27, 8, 59, 0, 0, wib) }
	r.tick(context.Background())
	if len(st.logs) != 0 {
		t.Fatal("belum jam kirim")
	}
	r.now = func() time.Time { return time.Date(2026, 10, 27, 9, 30, 0, 0, wib) }
	r.tick(context.Background())
	r.tick(context.Background()) // menit berikutnya tidak mengirim dua kali
	if len(st.logs) != 1 || st.logs[0].Status != "sent" || st.logs[0].ReportDate != "2026-10-26" {
		t.Fatalf("log = %+v", st.logs)
	}
	if !strings.Contains((*urls)[0], "/print/report-alert-summary?date=2026-10-26&shift=SHIFT_1#token=tok") {
		t.Fatalf("url cetak = %s", (*urls)[0])
	}
	r.now = func() time.Time { return time.Date(2026, 10, 28, 11, 30, 0, 0, wib) } // lewat jendela kejar 2 jam
	r.tick(context.Background())
	if len(st.logs) != 1 {
		t.Fatal("jadwal yang terlambat > 2 jam dilewati")
	}
}

func TestReportSendNowWithoutSMTP(t *testing.T) {
	r, st, _ := newTestReports(false)
	admin := access.Actor{UserID: uuid.New(), ProjectID: &st.sc.ProjectID, Role: access.RoleAdmin}
	l, err := r.SendNow(context.Background(), admin, st.sc.ID)
	if err == nil || l.Status != "failed" || !strings.Contains(*l.ErrorMessage, "SMTP") || len(st.logs) != 1 {
		t.Fatalf("log=%+v err=%v", l, err)
	}
	// Preview tetap bisa tanpa SMTP
	pdf, name, err := r.Preview(context.Background(), admin, st.sc.ID, "2026-10-26")
	if err != nil || string(pdf) != "%PDF" || name != "alert_summary_2026-10-26_shift_1.pdf" {
		t.Fatalf("preview %q %s %v", pdf, name, err)
	}
	viewer := access.Actor{UserID: uuid.New(), ProjectID: &st.sc.ProjectID, Role: access.RoleViewer}
	if _, err := r.SendNow(context.Background(), viewer, st.sc.ID); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("viewer: %v", err)
	}
}

func TestReportScheduleValidation(t *testing.T) {
	r, st, _ := newTestReports(true)
	admin := access.Actor{UserID: uuid.New(), ProjectID: &st.sc.ProjectID, Role: access.RoleAdmin}
	s := func(v string) *string { return &v }
	ids := []uuid.UUID{uuid.New()}
	off := 0
	for _, c := range []struct {
		in   domain.ReportScheduleInput
		want string
	}{
		{domain.ReportScheduleInput{Name: s(""), RecipientUserIDs: &ids}, "name"},
		{domain.ReportScheduleInput{Name: s("X"), SendTime: s("9:00"), RecipientUserIDs: &ids}, "send_time"},
		{domain.ReportScheduleInput{Name: s("X"), ShiftCode: s("SHIFT_9"), RecipientUserIDs: &ids}, "shift"},
		{domain.ReportScheduleInput{Name: s("X")}, "penerima"},
	} {
		if _, err := r.Save(context.Background(), admin, nil, c.in); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("mau galat %q, dapat %v", c.want, err)
		}
	}
	got, err := r.Save(context.Background(), admin, nil, domain.ReportScheduleInput{Name: s("OK"), ShiftCode: s(""), DateOffset: &off, RecipientUserIDs: &ids})
	if err != nil || got.ShiftCode != nil || got.DateOffset != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}
