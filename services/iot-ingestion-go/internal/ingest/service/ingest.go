// Package service = inti pipeline ingest: satu event dari alat (lewat HTTP atau TCP) dirapikan,
// divalidasi, diperkaya data induk, dinilai aturan domain, disimpan, lalu kondisi terakhir, kejadian,
// alert, siaran WebSocket, dan antrian WhatsApp diperbarui. Fitur lain dipakai lewat interface di bawah
// (dirakit di cmd/iot/main.go).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/repository"
)

// Result = hasil memproses satu event.
type Result struct {
	Inserted  bool              // baris baru tersimpan
	Duplicate bool              // event sudah pernah tersimpan → diabaikan
	LiveState *domain.LiveState // kondisi terakhir equipment yang baru (nil bila tidak berubah)
	Geofence  []domain.GeofenceEvent
	Fuel      []domain.FuelEvent
	Alerts    []domain.Alert
}

// Store = kemampuan database yang dibutuhkan service.
// *repository.Repository memenuhinya; test memakai versi palsu tanpa database.
type Store interface {
	InsertEvent(ctx context.Context, dev *devicedomain.Device, e *domain.Event) (repository.Saved, bool, error)
	InsertGeofenceEvent(ctx context.Context, g *domain.GeofenceEvent) error
	InsertAlert(ctx context.Context, a *domain.Alert) error
	CloseAlert(ctx context.Context, equipmentID uuid.UUID, alertType string, endedAt time.Time) (*domain.Alert, error)
	ExtendAlert(ctx context.Context, equipmentID uuid.UUID, alertType string, startedAt, endedAt time.Time) (*domain.Alert, error)
	InsertFuelEvent(ctx context.Context, f *domain.FuelEvent) error
}

// LiveStates = penyimpan kondisi terakhir per equipment (dipenuhi *service.Cache milik fitur livestate).
type LiveStates interface {
	Lock(equipmentID uuid.UUID) (unlock func())
	Get(equipmentID uuid.UUID) (domain.LiveState, bool)
	Apply(ctx context.Context, owner domain.Owner, e *domain.Event, telemetryID int64, receivedAt time.Time) (domain.LiveState, bool, error)
}

// Broadcaster = penyiar perubahan ke WebSocket (dipenuhi *ws.Hub milik fitur livestate).
type Broadcaster interface {
	Publish(projectID *uuid.UUID, msgType string, data any)
}

// Enricher = data induk core di memori (dipenuhi *masterdata.Cache): pengisi zona/shift/operator/breakdown,
// kalibrasi fuel, dan ambang alert efektif.
type Enricher interface {
	Enrich(projectID, equipmentID *uuid.UUID, e *domain.Event)
	Calibration(equipmentID *uuid.UUID) *domain.Calibration
	Rules(projectID *uuid.UUID, zoneName *string) domain.AlertRules
}

// AlertNotifier = antrian notifikasi alert (dipenuhi *service.Notifier milik fitur notification).
type AlertNotifier interface {
	AlertOpened(ctx context.Context, a domain.Alert)
}

// Jenis pesan siaran (sama dengan ws.Type* fitur livestate; ditulis ulang agar service tidak bergantung pada delivery).
const (
	msgEquipmentState = "equipment_state"
	msgGeofenceEvent  = "geofence_event"
	msgFuelEvent      = "fuel_event"
	msgAlert          = "alert"
)

// Ingest berisi aturan bisnis penerimaan data. Dipakai bersama oleh HTTP (Step 9) dan TCP (Step 12).
type Ingest struct {
	repo   Store
	live   LiveStates       // boleh nil (mis. di test yang tidak membutuhkannya)
	bus    Broadcaster      // boleh nil
	enrich Enricher         // boleh nil (tanpa data induk: kolom pengayaan kosong)
	notify AlertNotifier    // boleh nil (tanpa notifikasi WhatsApp)
	now    func() time.Time // bisa diganti saat test agar waktu "tetap"
}

func NewIngest(r Store, live LiveStates, bus Broadcaster) *Ingest {
	return &Ingest{repo: r, live: live, bus: bus, now: time.Now}
}

// WithEnricher memasang pengisi data induk (Fase C).
// WithNotifier memasang antrian notifikasi WhatsApp (Fase C5).
func (s *Ingest) WithNotifier(n AlertNotifier) *Ingest {
	s.notify = n
	return s
}

// notifyOpened memasukkan alert yang baru DIBUKA ke antrian notifikasi (tidak pernah menggagalkan).
func (s *Ingest) notifyOpened(ctx context.Context, a domain.Alert) {
	if s.notify != nil {
		s.notify.AlertOpened(ctx, a)
	}
}

func (s *Ingest) WithEnricher(en Enricher) *Ingest {
	s.enrich = en
	return s
}

// Handle: rapikan → cocokkan identitas → validasi → hitung status & isi data induk → simpan → perbarui kondisi terakhir.
// Error yang membungkus domain.ErrInvalid = kesalahan DATA (alat); error lain = kesalahan SERVER.
func (s *Ingest) Handle(ctx context.Context, dev *devicedomain.Device, e *domain.Event) (Result, error) {
	e.Normalize()

	// Identitas alat ditentukan oleh autentikasi (API key / IMEI login), bukan dipercaya dari isi data.
	if e.IMEI == "" {
		e.IMEI = dev.IMEI
	}
	if e.IMEI != dev.IMEI {
		err := fmt.Errorf("%w: imei di data (%s) tidak sesuai dengan alat yang login (%s)", domain.ErrInvalid, e.IMEI, dev.IMEI)
		s.logRejected(dev, e, err)
		return Result{}, err
	}

	if err := e.Validate(s.now()); err != nil {
		s.logRejected(dev, e, err)
		return Result{}, err
	}

	status := domain.OperationalStatus(e)
	e.EquipmentStatus = &status
	if s.enrich != nil {
		s.enrich.Enrich(dev.ProjectID, dev.EquipmentID, e) // dari memori, tanpa query (ADR 0009)
	}

	// Mulai di sini data SATU unit diproses berurutan: status muatan & kejadian bergantung pada kondisi
	// sebelumnya, jadi baca kondisi → simpan → perbarui kondisi tidak boleh diselingi data lain unit itu.
	var prev *domain.LiveState
	inOrder := true // false = data terlambat (lebih lama dari kondisi terakhir)
	if s.live != nil && dev.EquipmentID != nil {
		unlock := s.live.Lock(*dev.EquipmentID)
		defer unlock()
		if p, ok := s.live.Get(*dev.EquipmentID); ok {
			prev, inOrder = &p, !e.EventTime.Before(p.EventTime)
		}
	}
	// Breakdown MENEMPEL: unit tanpa catatan breakdown memakai nilai yang sedang berlaku di kondisi terakhirnya
	// (unit yang punya catatan sudah diisi pengayaan menurut status terbaru yang diinput). Tanpa keduanya: tidak breakdown.
	if e.IsBreakdown == nil {
		if prev != nil && prev.IsBreakdown != nil {
			e.IsBreakdown, e.BreakdownDescription = prev.IsBreakdown, prev.BreakdownDescription
		} else if e.ShiftCode != nil {
			no := false
			e.IsBreakdown = &no
		}
	}
	var cal *domain.Calibration
	rules := domain.DefaultAlertRules
	if s.enrich != nil {
		cal, rules = s.enrich.Calibration(dev.EquipmentID), s.enrich.Rules(dev.ProjectID, e.SegmentName)
	}
	var fuelOut []domain.FuelOutcome
	var alertOut []domain.AlertOutcome
	var geoOut []domain.GeofenceEvent
	if inOrder {
		domain.ApplyVessel(prev, e)
		var liters *float64 // hanya bacaan baru yang sah yang dinilai pendeteksi
		if domain.ApplyFuel(prev, cal, e) {
			liters = e.FuelVolumeLiter
		}
		var tr domain.Tracker
		if prev != nil {
			tr = prev.Tracker
		}
		tr, fuelOut = domain.DetectFuel(tr, e, liters, rules)
		tr, alertOut = domain.DetectAlerts(tr, e, rules)
		geoOut = domain.GeofenceTransitions(prev, e, &tr)
		e.Tracker = &tr
		e.AlertCount = alertCount(prev, e, fuelOut, alertOut)
	} else {
		// data terlambat: status muatan diperkirakan dari kondisi terakhir (tidak mengubahnya); selisih fuel & kejadian
		// dibiarkan kosong (urutannya tidak diketahui)
		domain.ApplyVesselLate(prev, e)
		domain.FuelLitersOnly(cal, e)
	}

	saved, inserted, err := s.repo.InsertEvent(ctx, dev, e)
	if err != nil {
		return Result{}, fmt.Errorf("simpan event: %w", err)
	}
	res := Result{Inserted: inserted, Duplicate: !inserted}
	if !inserted || s.live == nil || dev.EquipmentID == nil {
		return res, nil
	}

	owner := domain.Owner{EquipmentID: *dev.EquipmentID, ProjectID: dev.ProjectID, DeviceID: dev.ID}
	state, changed, err := s.live.Apply(ctx, owner, e, saved.ID, saved.ReceivedAt)
	if err != nil {
		// Data utama SUDAH tersimpan; kondisi terakhir akan terkoreksi oleh data berikutnya (±30 detik).
		// Jangan gagalkan request, supaya alat tidak mengirim ulang data yang sebenarnya sudah masuk.
		slog.Error("perbarui kondisi terakhir gagal", "equipment_id", owner.EquipmentID, "err", err)
		return res, nil
	}
	if changed {
		res.LiveState = &state
		s.publish(state.ProjectID, msgEquipmentState, state) // peta live langsung bergerak
	}
	if inOrder {
		res.Geofence = s.geofenceEvents(ctx, owner, geoOut, e, saved.ID)
		res.Fuel, res.Alerts = s.fuelEvents(ctx, owner, e, saved.ID, fuelOut, rules)
		res.Alerts = append(res.Alerts, s.durationAlerts(ctx, owner, e, saved.ID, alertOut, rules)...)
	}
	return res, nil
}

// alertCount = jumlah alert unit pada tanggal operasional + shift data ini: lanjut dari kondisi sebelumnya bila
// masih shift yang sama, mulai dari 0 bila shift berganti; ditambah alert yang dibuka oleh data ini.
func alertCount(prev *domain.LiveState, e *domain.Event, fuel []domain.FuelOutcome, alerts []domain.AlertOutcome) *int {
	n := 0
	if prev != nil && strEq(prev.OperationalDate, e.OperationalDate) && strEq(prev.ShiftCode, e.ShiftCode) {
		n = prev.AlertCount
	}
	for _, f := range fuel {
		if f.AlertType != "" && !f.Extend {
			n++
		}
	}
	for _, a := range alerts {
		if a.Action == domain.AlertOpen {
			n++
		}
	}
	return &n
}

func strEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// durationAlerts membuat / memperpanjang / menutup alert berdurasi (speed, off-track) dan menyiarkannya.
// Alert yang ternyata masih terbuka di database (mis. keadaan pendeteksi hilang) tidak dibuat dobel.
func (s *Ingest) durationAlerts(ctx context.Context, owner domain.Owner, e *domain.Event, telemetryID int64,
	outs []domain.AlertOutcome, rules domain.AlertRules) []domain.Alert {

	var alerts []domain.Alert
	for _, o := range outs {
		switch o.Action {
		case domain.AlertExtend:
			if a := s.extendAlert(ctx, owner, o.AlertType, o.StartedAt, *o.EndedAt); a != nil {
				alerts = append(alerts, *a)
			}
		case domain.AlertOpen:
			a := domain.Alert{ProjectID: owner.ProjectID, EquipmentID: owner.EquipmentID, DeviceID: owner.DeviceID,
				TelemetryEventID: telemetryID, AlertType: o.AlertType, StartedAt: o.StartedAt, EndedAt: o.EndedAt,
				OperationalDate: e.OperationalDate, ShiftCode: e.ShiftCode, ThresholdValue: o.Threshold,
				Snapshot: domain.SnapshotOf(e), OperatorName: e.OperatorName, Recipients: rules[o.AlertType].Recipients}
			err := s.repo.InsertAlert(ctx, &a)
			if errors.Is(err, domain.ErrAlertAlreadyOpen) {
				continue
			}
			if err != nil {
				slog.Error("buka alert gagal", "equipment_id", owner.EquipmentID, "jenis", o.AlertType, "err", err)
				continue
			}
			alerts = append(alerts, a)
			s.publish(owner.ProjectID, msgAlert, a)
			s.notifyOpened(ctx, a)
		case domain.AlertClose:
			a, err := s.repo.CloseAlert(ctx, owner.EquipmentID, o.AlertType, *o.EndedAt)
			if err != nil {
				slog.Error("tutup alert gagal", "equipment_id", owner.EquipmentID, "jenis", o.AlertType, "err", err)
				continue
			}
			if a != nil {
				alerts = append(alerts, *a)
				s.publish(owner.ProjectID, msgAlert, *a)
			}
		}
	}
	return alerts
}

// fuelEvents menyimpan kejadian fuel (dan alert fuel bila ada), lalu menyiarkannya.
// Alert fuel = kejadian sesaat: started_at = awal periode (berhenti / mesin mati), ended_at = saat terdeteksi.
func (s *Ingest) fuelEvents(ctx context.Context, owner domain.Owner, e *domain.Event, telemetryID int64,
	outs []domain.FuelOutcome, rules domain.AlertRules) ([]domain.FuelEvent, []domain.Alert) {

	var events []domain.FuelEvent
	var alerts []domain.Alert
	snap := domain.SnapshotOf(e)
	for _, o := range outs {
		if o.Extend { // alert FUD yang sama, fuel masih turun: perpanjang akhirnya (aturan lama)
			if a := s.extendAlert(ctx, owner, o.AlertType, o.StartedAt, e.EventTime); a != nil {
				alerts = append(alerts, *a)
			}
			continue
		}
		var alertID *int64
		if o.AlertType != "" {
			end, threshold := e.EventTime, o.Threshold
			// selisih fuel di alert = selisih data pemicu (sama dengan lama); kumulatif ada di fuel_events
			a := domain.Alert{ProjectID: owner.ProjectID, EquipmentID: owner.EquipmentID, DeviceID: owner.DeviceID,
				TelemetryEventID: telemetryID, AlertType: o.AlertType, StartedAt: o.StartedAt, EndedAt: &end,
				OperationalDate: e.OperationalDate, ShiftCode: e.ShiftCode, ThresholdValue: &threshold,
				Snapshot: snap, OperatorName: e.OperatorName, Recipients: rules[o.AlertType].Recipients}
			if err := s.repo.InsertAlert(ctx, &a); err != nil {
				slog.Error("simpan alert fuel gagal", "equipment_id", owner.EquipmentID, "jenis", o.AlertType, "err", err)
			} else {
				alertID = &a.ID
				alerts = append(alerts, a)
				s.publish(owner.ProjectID, msgAlert, a)
				s.notifyOpened(ctx, a)
			}
		}
		fe := domain.FuelEvent{ProjectID: owner.ProjectID, EquipmentID: owner.EquipmentID, DeviceID: owner.DeviceID,
			TelemetryEventID: telemetryID, EventType: o.EventType, EventTime: e.EventTime,
			OperationalDate: e.OperationalDate, ShiftCode: e.ShiftCode, FuelDeltaLiter: o.DeltaLiter,
			Description: o.Description, AlertID: alertID, Snapshot: snap}
		if err := s.repo.InsertFuelEvent(ctx, &fe); err != nil {
			slog.Error("simpan kejadian fuel gagal", "equipment_id", owner.EquipmentID, "jenis", o.EventType, "err", err)
			continue
		}
		events = append(events, fe)
		s.publish(owner.ProjectID, msgFuelEvent, fe)
	}
	return events, alerts
}

// extendAlert memperpanjang ended_at alert (unit + jenis + awal sama) dan menyiarkannya.
func (s *Ingest) extendAlert(ctx context.Context, owner domain.Owner, alertType string, startedAt, endedAt time.Time) *domain.Alert {
	a, err := s.repo.ExtendAlert(ctx, owner.EquipmentID, alertType, startedAt, endedAt)
	if err != nil {
		slog.Error("perpanjang alert gagal", "equipment_id", owner.EquipmentID, "jenis", alertType, "err", err)
		return nil
	}
	if a != nil {
		s.publish(owner.ProjectID, msgAlert, *a)
	}
	return a
}

// geofenceEvents menyimpan & menyiarkan kejadian masuk/keluar area. Gagal simpan hanya dicatat di log:
// data utama sudah tersimpan dan tidak boleh dikirim ulang oleh alat.
func (s *Ingest) geofenceEvents(ctx context.Context, owner domain.Owner, events []domain.GeofenceEvent, e *domain.Event, telemetryID int64) []domain.GeofenceEvent {
	var out []domain.GeofenceEvent
	for _, g := range events {
		g.ProjectID, g.EquipmentID, g.DeviceID, g.TelemetryEventID = owner.ProjectID, owner.EquipmentID, owner.DeviceID, telemetryID
		g.EventTime, g.OperationalDate, g.ShiftCode = e.EventTime, e.OperationalDate, e.ShiftCode
		g.Latitude, g.Longitude, g.SpeedKph, g.IsEngineOn = e.Latitude, e.Longitude, e.SpeedKph, e.IsEngineOn
		g.EquipmentStatus, g.VesselStatus, g.FuelLevelRaw, g.MileageM = e.EquipmentStatus, e.VesselStatus, e.FuelLevelRaw, e.MileageM
		if err := s.repo.InsertGeofenceEvent(ctx, &g); err != nil {
			slog.Error("simpan kejadian zona gagal", "equipment_id", owner.EquipmentID, "jenis", g.EventType, "err", err)
			continue
		}
		s.publish(owner.ProjectID, msgGeofenceEvent, g)
		out = append(out, g)
	}
	return out
}

func (s *Ingest) publish(projectID *uuid.UUID, msgType string, data any) {
	if s.bus != nil {
		s.bus.Publish(projectID, msgType, data)
	}
}

// logRejected mencatat event yang ditolak (keputusan kita: cukup di log aplikasi, tanpa tabel).
func (s *Ingest) logRejected(dev *devicedomain.Device, e *domain.Event, err error) {
	slog.Warn("event ditolak",
		"device_id", dev.ID,
		"imei", dev.IMEI,
		"event_id", e.EventID,
		"event_time", e.EventTime,
		"alasan", err.Error(),
	)
}

// IsInvalid membantu delivery (HTTP/TCP) membedakan kesalahan data dari kesalahan server.
func IsInvalid(err error) bool { return errors.Is(err, domain.ErrInvalid) }
