// Package masterdata menyimpan DATA INDUK dari core (project, shift, equipment, operator, breakdown)
// di memori layanan IoT (ADR 0005/0009): dibaca dari view core.v_iot_* saat start, lalu dimuat ulang
// PER BAGIAN saat core mengirim NOTIFY "simha_master_changed", plus muat ulang penuh berkala.
// Pengayaan telemetry tidak pernah membaca database per paket.
package masterdata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// Bagian data induk; masing-masing dimuat ulang sendiri.
const (
	PartProjects     = "projects"     // project + shift
	PartEquipment    = "equipment"    // kode unit
	PartOperators    = "operators"    // operator harian (7 hari terakhir)
	PartBreakdowns   = "breakdowns"   // breakdown harian (7 hari terakhir)
	PartZones        = "zones"        // zona peta (berlaku + 8 hari terakhir, untuk data terlambat)
	PartCalibrations = "calibrations" // kalibrasi fuel per equipment
	PartAlertRules   = "alert_rules"  // ambang alert efektif per project + zona (+ penerima)
	PartRecipients   = "recipients"   // nama & nomor HP penerima notifikasi
)

var allParts = []string{PartProjects, PartEquipment, PartOperators, PartBreakdowns, PartZones, PartCalibrations, PartAlertRules, PartRecipients}

// partsOfTable = tabel core yang berubah (payload NOTIFY) → bagian yang perlu dimuat ulang.
// Tabel lain diabaikan.
var partsOfTable = map[string][]string{
	"core.projects":                {PartProjects, PartAlertRules},
	"core.shifts":                  {PartProjects},
	"core.equipment":               {PartEquipment},
	"core.operator_assignments":    {PartOperators},
	"core.equipment_breakdowns":    {PartBreakdowns},
	"geo.zones":                    {PartZones, PartAlertRules}, // aturan efektif dihitung per nama zona
	"core.fuel_calibrations":       {PartCalibrations},
	"core.fuel_calibration_points": {PartCalibrations},
	"core.alert_types":             {PartAlertRules},
	"core.alert_rules":             {PartAlertRules},
	"core.alert_rule_recipients":   {PartAlertRules},
	"auth.users":                   {PartAlertRules, PartRecipients}, // penerima nonaktif / nomor HP berubah
	"auth.project_members":         {PartAlertRules},
}

// ---------------------------------------------------------------------
// Baris dari view (diisi repository)
// ---------------------------------------------------------------------

type ShiftRow struct {
	ProjectID          uuid.UUID
	ProjectCode        string
	ProjectName        string
	Timezone           string
	Code, Name         string
	StartTime, EndTime string // HH:MM
	Sequence           int
}

type EquipmentRow struct {
	ID, ProjectID uuid.UUID
	Code          string
}

type OperatorRow struct {
	ProjectID       uuid.UUID
	OperationalDate string // YYYY-MM-DD
	ShiftCode       string
	EquipmentCode   string
	OperatorName    string
}

type BreakdownRow struct {
	ProjectID          uuid.UUID
	OperationalDate    string
	ShiftCode          string
	EquipmentCode      string
	Status             string // breakdown | continue | ready
	StartedAt, EndedAt *time.Time
	Description        *string
	UpdatedAt          time.Time
}

type ZoneRow struct {
	ID, ProjectID uuid.UUID
	SourceFID     *int32
	Category      string
	Name          string
	Priority      int
	AreaM2        float64
	GeoJSON       string // ST_AsGeoJSON(geom)
	ValidFrom     time.Time
	ValidTo       *time.Time
}

type CalibrationRow struct {
	EquipmentID    uuid.UUID
	Type           string
	CapacityLiters float64
	Sensor, Liters float64
}

// AlertRuleRow = ambang efektif satu jenis alert untuk project + zona (ZoneName nil = tingkat project).
type AlertRuleRow struct {
	ProjectID uuid.UUID
	ZoneName  *string
	AlertType string
	domain.AlertRule
}

// Recipient = penerima notifikasi (dari core.v_iot_alert_recipients: user aktif yang punya nomor HP).
type Recipient struct {
	UserID uuid.UUID
	Name   string
	Phone  string
}

// Store = pembaca view core (dipenuhi *Repository di repository.go).
type Store interface {
	LoadShifts(ctx context.Context) ([]ShiftRow, error)
	LoadEquipment(ctx context.Context) ([]EquipmentRow, error)
	LoadOperators(ctx context.Context) ([]OperatorRow, error)
	LoadBreakdowns(ctx context.Context) ([]BreakdownRow, error)
	LoadZones(ctx context.Context) ([]ZoneRow, error)
	LoadCalibrations(ctx context.Context) ([]CalibrationRow, error)
	LoadAlertRules(ctx context.Context) ([]AlertRuleRow, error)
	LoadRecipients(ctx context.Context) ([]Recipient, error)
}

// ---------------------------------------------------------------------
// Snapshot: data induk yang TIDAK diubah setelah dibuat (aman dibaca banyak goroutine tanpa kunci)
// ---------------------------------------------------------------------

type project struct {
	code, name string
	loc        *time.Location
	shifts     []domain.Shift
}

// dayKey = project + tanggal operasional + shift + kode unit.
type dayKey struct {
	project       uuid.UUID
	date, shift   string
	equipmentCode string
}

type snapshot struct {
	projects       map[uuid.UUID]project
	equipment      map[uuid.UUID]string // equipment_id → kode unit
	operators      map[dayKey]string
	breakdowns     map[dayKey][]BreakdownRow                  // per tanggal+shift+unit
	unitBreakdowns map[unitKey][]BreakdownRow                 // per unit, yang terbaru diinput di depan (status menempel)
	zones          map[uuid.UUID][]domain.Zone                // per project
	calibrations   map[uuid.UUID]*domain.Calibration          // per equipment
	rules          map[uuid.UUID]map[string]domain.AlertRules // project → nama zona (huruf kecil; "" = project)
	recipients     map[uuid.UUID]Recipient                    // user_id → nama & nomor HP
	loadedAt       time.Time
}

// Cache = data induk di memori. Nilai nol tidak dipakai; buat dengan New.
type Cache struct {
	store  Store
	snap   atomic.Pointer[snapshot]
	reload sync.Mutex // muat ulang satu per satu

	hookMu   sync.Mutex
	onReload []func(parts []string)
}

// OnReload mendaftarkan fungsi yang dipanggil SETIAP KALI data induk berhasil dimuat ulang (setelah NOTIFY dari core
// atau muat ulang berkala; bukan pada muat awal). Dipakai untuk memperbarui kondisi terakhir unit saat breakdown / operator
// berubah. Fungsi harus cepat dan tidak memblokir (mis. mengirim sinyal ke saluran).
func (c *Cache) OnReload(f func(parts []string)) {
	c.hookMu.Lock()
	defer c.hookMu.Unlock()
	c.onReload = append(c.onReload, f)
}

func New(store Store) *Cache {
	c := &Cache{store: store}
	c.snap.Store(&snapshot{projects: map[uuid.UUID]project{}, equipment: map[uuid.UUID]string{},
		operators: map[dayKey]string{}, breakdowns: map[dayKey][]BreakdownRow{}, unitBreakdowns: map[unitKey][]BreakdownRow{}, zones: map[uuid.UUID][]domain.Zone{},
		calibrations: map[uuid.UUID]*domain.Calibration{}, rules: map[uuid.UUID]map[string]domain.AlertRules{},
		recipients: map[uuid.UUID]Recipient{}})
	return c
}

// Load memuat SEMUA bagian.
func (c *Cache) Load(ctx context.Context) error { return c.Reload(ctx, allParts...) }

// Reload memuat ulang bagian tertentu. Snapshot baru dibuat dari salinan yang lama (bagian lain tetap),
// lalu ditukar sekaligus → pembaca tidak pernah melihat data setengah jadi.
func (c *Cache) Reload(ctx context.Context, parts ...string) error {
	c.reload.Lock()
	defer c.reload.Unlock()
	old := c.snap.Load()
	next := *old
	var errs []error
	for _, p := range parts {
		var err error
		switch p {
		case PartProjects:
			err = keep(&next.projects, c.loadProjects, ctx)
		case PartEquipment:
			err = keep(&next.equipment, c.loadEquipment, ctx)
		case PartOperators:
			err = keep(&next.operators, c.loadOperators, ctx)
		case PartBreakdowns:
			if err = keep(&next.breakdowns, c.loadBreakdowns, ctx); err == nil {
				next.unitBreakdowns = breakdownsByUnit(next.breakdowns)
			}
		case PartZones:
			err = keep(&next.zones, c.loadZones, ctx)
		case PartCalibrations:
			err = keep(&next.calibrations, c.loadCalibrations, ctx)
		case PartAlertRules:
			err = keep(&next.rules, c.loadAlertRules, ctx)
		case PartRecipients:
			err = keep(&next.recipients, c.loadRecipients, ctx)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
		}
	}
	next.loadedAt = time.Now()
	c.snap.Store(&next)
	return errors.Join(errs...)
}

// keep mengganti *dst dengan hasil load HANYA bila berhasil; gagal → data lama tetap dipakai.
func keep[T any](dst *T, load func(context.Context) (T, error), ctx context.Context) error {
	v, err := load(ctx)
	if err == nil {
		*dst = v
	}
	return err
}

func (c *Cache) loadProjects(ctx context.Context) (map[uuid.UUID]project, error) {
	rows, err := c.store.LoadShifts(ctx)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]project{}
	for _, r := range rows {
		p, ok := out[r.ProjectID]
		if !ok {
			loc, err := time.LoadLocation(r.Timezone)
			if err != nil {
				slog.Warn("zona waktu project tidak dikenal, shift project dilewati", "project", r.ProjectCode, "timezone", r.Timezone)
				continue
			}
			p.loc, p.code, p.name = loc, r.ProjectCode, r.ProjectName
		}
		start, err1 := domain.ParseClock(r.StartTime)
		end, err2 := domain.ParseClock(r.EndTime)
		if err1 != nil || err2 != nil {
			slog.Warn("jam shift tidak valid, dilewati", "project", r.ProjectCode, "shift", r.Code)
			continue
		}
		p.shifts = append(p.shifts, domain.Shift{Code: r.Code, Name: r.Name, StartMin: start, EndMin: end, Sequence: r.Sequence})
		out[r.ProjectID] = p
	}
	return out, nil
}

func (c *Cache) loadEquipment(ctx context.Context) (map[uuid.UUID]string, error) {
	rows, err := c.store.LoadEquipment(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		out[r.ID] = strings.ToUpper(r.Code)
	}
	return out, nil
}

func (c *Cache) loadOperators(ctx context.Context) (map[dayKey]string, error) {
	rows, err := c.store.LoadOperators(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[dayKey]string, len(rows))
	for _, r := range rows {
		out[dayKey{r.ProjectID, r.OperationalDate, r.ShiftCode, strings.ToUpper(r.EquipmentCode)}] = r.OperatorName
	}
	return out, nil
}

func (c *Cache) loadBreakdowns(ctx context.Context) (map[dayKey][]BreakdownRow, error) {
	rows, err := c.store.LoadBreakdowns(ctx)
	if err != nil {
		return nil, err
	}
	out := map[dayKey][]BreakdownRow{}
	for _, r := range rows {
		k := dayKey{r.ProjectID, r.OperationalDate, r.ShiftCode, strings.ToUpper(r.EquipmentCode)}
		out[k] = append(out[k], r)
	}
	return out, nil
}

func (c *Cache) loadZones(ctx context.Context) (map[uuid.UUID][]domain.Zone, error) {
	rows, err := c.store.LoadZones(ctx)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID][]domain.Zone{}
	for _, r := range rows {
		z := domain.Zone{ID: r.ID, ProjectID: r.ProjectID, SourceFID: r.SourceFID, Category: r.Category, Name: r.Name,
			Priority: r.Priority, AreaM2: r.AreaM2, ValidFrom: r.ValidFrom, ValidTo: r.ValidTo}
		if err := z.SetGeoJSON(r.GeoJSON); err != nil {
			slog.Warn("geometri zona tidak terbaca, zona dilewati", "zone_id", r.ID, "name", r.Name, "err", err)
			continue
		}
		out[r.ProjectID] = append(out[r.ProjectID], z)
	}
	return out, nil
}

func (c *Cache) loadCalibrations(ctx context.Context) (map[uuid.UUID]*domain.Calibration, error) {
	rows, err := c.store.LoadCalibrations(ctx)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]*domain.Calibration{}
	for _, r := range rows { // repository mengurutkan per equipment & nilai sensor
		cal, ok := out[r.EquipmentID]
		if !ok {
			cal = &domain.Calibration{Type: r.Type, CapacityLiters: r.CapacityLiters}
			out[r.EquipmentID] = cal
		}
		cal.Points = append(cal.Points, domain.CalPoint{Sensor: r.Sensor, Liters: r.Liters})
	}
	return out, nil
}

func (c *Cache) loadAlertRules(ctx context.Context) (map[uuid.UUID]map[string]domain.AlertRules, error) {
	rows, err := c.store.LoadAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]map[string]domain.AlertRules{}
	for _, r := range rows {
		zone := ""
		if r.ZoneName != nil {
			zone = strings.ToLower(*r.ZoneName)
		}
		if out[r.ProjectID] == nil {
			out[r.ProjectID] = map[string]domain.AlertRules{}
		}
		if out[r.ProjectID][zone] == nil {
			out[r.ProjectID][zone] = domain.AlertRules{}
		}
		out[r.ProjectID][zone][r.AlertType] = r.AlertRule
	}
	return out, nil
}

// Calibration = kalibrasi fuel equipment (nil = belum dikalibrasi).
func (c *Cache) Calibration(equipmentID *uuid.UUID) *domain.Calibration {
	if equipmentID == nil {
		return nil
	}
	return c.snap.Load().calibrations[*equipmentID]
}

// Rules = ambang alert efektif untuk project + zona (aturan zona → project → bawaan jenis).
// Data induk belum termuat → bawaan sistem lama (DefaultAlertRules).
func (c *Cache) Rules(projectID *uuid.UUID, zoneName *string) domain.AlertRules {
	if projectID == nil {
		return domain.DefaultAlertRules
	}
	byZone := c.snap.Load().rules[*projectID]
	if zoneName != nil {
		if r, ok := byZone[strings.ToLower(*zoneName)]; ok {
			return r
		}
	}
	if r, ok := byZone[""]; ok {
		return r
	}
	return domain.DefaultAlertRules
}

// ---------------------------------------------------------------------
// Pengayaan
// ---------------------------------------------------------------------

// Enrich mengisi zona, shift, tanggal operasional, operator, dan breakdown pada event (bila data induk ada).
// Tidak pernah gagal: data induk yang belum ada → kolom dibiarkan kosong.
func (c *Cache) Enrich(projectID, equipmentID *uuid.UUID, e *domain.Event) {
	if projectID == nil {
		return
	}
	s := c.snap.Load()
	enrichZone(s.zones[*projectID], e)
	p, ok := s.projects[*projectID]
	if !ok {
		return
	}
	date, shift, ok := domain.ResolvePeriod(p.shifts, p.loc, e.EventTime)
	if !ok {
		return
	}
	e.OperationalDate, e.ShiftCode = &date, &shift.Code

	if equipmentID == nil {
		return
	}
	code, ok := s.equipment[*equipmentID]
	if !ok {
		return
	}
	k := dayKey{*projectID, date, shift.Code, code}
	if name, ok := s.operators[k]; ok {
		e.OperatorName = &name
	}
	// Breakdown MENEMPEL: ikut status terbaru yang diinput untuk unit ini (lihat stickyBreakdown). Unit tanpa catatan →
	// dibiarkan kosong; pengisiannya (nilai yang sedang berlaku / false) dilakukan service ingest.
	if is, desc, known := stickyBreakdown(s.unitBreakdowns[unitKey{*projectID, code}]); known {
		e.IsBreakdown, e.BreakdownDescription = &is, desc
	}
}

// enrichZone mengisi zona tempat titik berada (peta yang berlaku pada jam data).
// Tanpa posisi, atau project belum punya peta → zona TIDAK diketahui (kolom kosong), bukan "di luar":
// supaya alert off-track tidak menyala hanya karena peta belum diunggah.
func enrichZone(zones []domain.Zone, e *domain.Event) {
	if e.Latitude == nil || e.Longitude == nil || len(zones) == 0 {
		return
	}
	inside := false
	var fid int32
	if z := domain.FindZone(zones, *e.Longitude, *e.Latitude, e.EventTime); z != nil {
		inside = true
		if z.SourceFID != nil {
			fid = *z.SourceFID
		}
		id, cat, name := z.ID, z.Category, z.Name
		e.ZoneID, e.LocationCategory, e.SegmentName = &id, &cat, &name
	}
	e.IsInsideGeofence, e.GeofenceFID = &inside, &fid // di luar semua zona: fid 0 (sama dengan lama)
}

// Status = ringkasan isi cache (untuk log & health).
type Status struct {
	Projects, Equipment, Operators, Breakdowns, Zones int
	LoadedAt                                          time.Time
}

func (c *Cache) Status() Status {
	s := c.snap.Load()
	n, z := 0, 0
	for _, b := range s.breakdowns {
		n += len(b)
	}
	for _, zs := range s.zones {
		z += len(zs)
	}
	return Status{len(s.projects), len(s.equipment), len(s.operators), n, z, s.loadedAt}
}

// ---------------------------------------------------------------------
// Pendengar NOTIFY
// ---------------------------------------------------------------------

// Notifier = sumber NOTIFY (dipenuhi *Listener di repository.go). Next menunggu satu pesan; menyambung
// ulang sendiri bila koneksi putus (mengembalikan error, lalu dipanggil lagi).
type Notifier interface {
	Next(ctx context.Context) (payload string, err error)
}

// Run mendengarkan perubahan data induk sampai ctx berakhir:
//   - NOTIFY masuk → kumpulkan sebentar (debounce) → muat ulang bagian terkait saja
//   - tidak ada NOTIFY selama `every` → muat ulang semua (cadangan, mis. view 7 hari bergeser)
//   - koneksi putus → tunggu, muat ulang semua (bisa ada NOTIFY terlewat), sambung lagi
func (c *Cache) Run(ctx context.Context, n Notifier, every, debounce time.Duration) {
	backoff := time.Second
	next := time.Now().Add(every)
	for ctx.Err() == nil {
		wait, cancel := context.WithDeadline(ctx, next)
		payload, err := n.Next(wait)
		cancel()
		switch {
		case ctx.Err() != nil:
			return
		case err != nil && wait.Err() != nil: // batas waktu cadangan
			c.reloadLogged(ctx, "berkala", allParts...)
			next = time.Now().Add(every)
		case err != nil:
			slog.Warn("pendengar data induk terputus, menyambung ulang", "err", err, "jeda", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 30*time.Second)
			c.reloadLogged(ctx, "sambung ulang", allParts...)
		default:
			backoff = time.Second
			parts := map[string]bool{}
			addPart(parts, payload)
			deadline := time.Now().Add(debounce) // satu impor Excel = banyak NOTIFY → satu kali muat
			for {
				dctx, dcancel := context.WithDeadline(ctx, deadline)
				p, err := n.Next(dctx)
				dcancel()
				if err != nil {
					break
				}
				addPart(parts, p)
			}
			if len(parts) > 0 {
				list := make([]string, 0, len(parts))
				for p := range parts {
					list = append(list, p)
				}
				c.reloadLogged(ctx, "notify", list...)
			}
		}
	}
}

func addPart(parts map[string]bool, table string) {
	for _, p := range partsOfTable[table] {
		parts[p] = true
	}
}

func (c *Cache) reloadLogged(ctx context.Context, why string, parts ...string) {
	if err := c.Reload(ctx, parts...); err != nil {
		slog.Error("muat ulang data induk gagal (data lama tetap dipakai)", "alasan", why, "err", err)
		return
	}
	c.hookMu.Lock()
	hooks := append([]func([]string){}, c.onReload...)
	c.hookMu.Unlock()
	for _, h := range hooks {
		h(parts)
	}
	st := c.Status()
	slog.Info("data induk dimuat ulang", "alasan", why, "bagian", parts, "project", st.Projects,
		"equipment", st.Equipment, "operator", st.Operators, "breakdown", st.Breakdowns, "zona", st.Zones)
}

func (c *Cache) loadRecipients(ctx context.Context) (map[uuid.UUID]Recipient, error) {
	rows, err := c.store.LoadRecipients(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Recipient, len(rows))
	for _, r := range rows {
		out[r.UserID] = r
	}
	return out, nil
}

// Recipients = penerima yang masih aktif & punya nomor HP di antara ids (urutan ids dipertahankan).
func (c *Cache) Recipients(ids []uuid.UUID) []Recipient {
	s := c.snap.Load()
	var out []Recipient
	for _, id := range ids {
		if r, ok := s.recipients[id]; ok {
			out = append(out, r)
		}
	}
	return out
}

// Label = kode unit, nama project, dan zona waktu project (untuk isi pesan notifikasi).
func (c *Cache) Label(projectID, equipmentID *uuid.UUID) (equipmentCode, projectName string, loc *time.Location) {
	s := c.snap.Load()
	if equipmentID != nil {
		equipmentCode = s.equipment[*equipmentID]
	}
	if projectID != nil {
		if p, ok := s.projects[*projectID]; ok {
			projectName, loc = p.name, p.loc
		}
	}
	return equipmentCode, projectName, loc
}
