// Package domain = bentuk data package domain aturan murni fitur notification: antrian kiriman WhatsApp (outbox),
// format nomor HP, dan isi pesan alert. Alert sendiri berasal dari pipeline ingest (ingest/domain).
package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	ingestdomain "simha/iot-ingestion/internal/ingest/domain"
)

// Status notifikasi (iot.notifications).
const (
	NotifyPending = "pending"
	NotifySent    = "sent"
	NotifyFailed  = "failed"
)

// Notification = satu kiriman di antrian (outbox).
type Notification struct {
	ID              int64
	AlertID         int64
	ProjectID       *uuid.UUID
	RecipientUserID *uuid.UUID
	Phone           string
	Message         string
	Attempts        int
}

// NormalizePhone = format yang diterima server WhatsApp: kode negara tanpa "+" (62xxx) — aturan sistem lama.
// "" bila tidak ada angka.
func NormalizePhone(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)
	switch {
	case digits == "":
		return ""
	case strings.HasPrefix(digits, "62"):
		return digits
	case strings.HasPrefix(digits, "0"):
		return "62" + digits[1:]
	case strings.HasPrefix(digits, "8"):
		return "62" + digits
	}
	return digits
}

// AlertNames = nama tampilan jenis alert (sama dengan core.alert_types.name).
var AlertNames = map[string]string{
	ingestdomain.AlertOffTrack:              "Off Track",
	ingestdomain.AlertOverspeed:             "Overspeed",
	ingestdomain.AlertUnderspeed:            "Underspeed",
	ingestdomain.AlertFuelDecreaseEngineOn:  "Fuel Decrease Engine On",
	ingestdomain.AlertFuelDecreaseEngineOff: "Fuel Decrease Engine Off",
}

// AlertMessage = isi pesan WhatsApp sebuah alert. Format mengikuti pesan sistem lama; jam memakai zona waktu
// project (lama: selalu WIB) dan nilai ambang ikut ditampilkan.
func AlertMessage(a ingestdomain.Alert, equipmentCode, projectName string, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	name := AlertNames[a.AlertType]
	if name == "" {
		name = a.AlertType
	}
	place := projectName
	if a.SegmentName != nil && *a.SegmentName != "" {
		place = strings.Trim(place+" / "+*a.SegmentName, " /")
	}
	shift := ""
	if a.ShiftCode != nil {
		shift = " (" + *a.ShiftCode + ")"
	}
	t := a.StartedAt.In(loc)
	lines := []string{
		"⚠️ SIMHA ALERT – " + name,
		"Equipment : " + orDash(equipmentCode),
		"Project   : " + orDash(place),
		"Speed     : " + fmtNum(a.SpeedKph, "km/h"),
		"Fuel      : " + fmtNum(a.FuelVolumeLiter, "liter"),
	}
	if a.FuelDeltaLiter != nil && strings.HasPrefix(a.AlertType, "fuel_") {
		lines = append(lines, "Turun     : "+fmtNum(ptrAbs(*a.FuelDeltaLiter), "liter"))
	}
	if a.ThresholdValue != nil {
		lines = append(lines, "Ambang    : "+thresholdText(a.AlertType, *a.ThresholdValue))
	}
	lines = append(lines,
		"Operator  : "+orDash(deref(a.OperatorName)),
		"Waktu     : "+t.Format("02-01-2006 15:04")+" "+t.Format("MST")+shift,
	)
	if a.Latitude != nil && a.Longitude != nil {
		lines = append(lines, fmt.Sprintf("Lokasi    : https://maps.google.com/?q=%v,%v", *a.Latitude, *a.Longitude))
	}
	return strings.Join(lines, "\n")
}

func thresholdText(alertType string, v float64) string {
	switch alertType {
	case ingestdomain.AlertOverspeed, ingestdomain.AlertUnderspeed:
		return fmt.Sprintf("%g km/h", v)
	case ingestdomain.AlertOffTrack:
		return fmt.Sprintf("%g menit", v)
	}
	return fmt.Sprintf("%g liter", v)
}

func fmtNum(p *float64, unit string) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%g %s", round1(*p), unit)
}

func round1(v float64) float64 { return float64(int64(v*10+sign(v)*0.5)) / 10 }

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func ptrAbs(v float64) *float64 {
	if v < 0 {
		v = -v
	}
	return &v
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
