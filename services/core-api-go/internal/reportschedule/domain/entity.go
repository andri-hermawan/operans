// Package domain (fitur reportschedule) = jadwal kirim laporan PDF lewat email + log pengirimannya.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReportTypes = laporan yang bisa dijadwalkan → halaman cetak frontend yang dirender menjadi PDF.
// Menambah laporan baru = tambah satu entri di sini + satu halaman /print/... di frontend.
var ReportTypes = map[string]ReportType{
	"alert_summary":         {Label: "Alert Summary", PrintPath: "/print/report-alert-summary"},
	"executive_hauling":     {Label: "Executive Summary Report Hauling", PrintPath: "/print/report-executive-hauling"},
	"executive_emplacement": {Label: "Executive Summary Report Emplacement", PrintPath: "/print/report-executive-emplacement"},
}

type ReportType struct {
	Label     string `json:"label"`
	PrintPath string `json:"-"`
}

// ReportRecipient = penerima email laporan.
type ReportRecipient struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
}

type ReportSchedule struct {
	ID               uuid.UUID         `json:"id"`
	ProjectID        uuid.UUID         `json:"project_id"`
	Name             string            `json:"name"`
	ReportType       string            `json:"report_type"`
	ShiftCode        *string           `json:"shift_code"` // nil = satu hari operasional
	DateOffset       int               `json:"date_offset"`
	SendTime         string            `json:"send_time"` // HH:MM jam lokal project
	RecipientUserIDs []uuid.UUID       `json:"recipient_user_ids"`
	Recipients       []ReportRecipient `json:"recipients"` // hanya user aktif
	IsActive         bool              `json:"is_active"`
	LastRunAt        *time.Time        `json:"last_run_at"`
	Timezone         string            `json:"timezone"`
	CreatedBy        *uuid.UUID        `json:"-"`
	UpdatedBy        *uuid.UUID        `json:"-"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type ReportScheduleInput struct {
	ProjectID        *uuid.UUID   `json:"project_id"` // saat tambah (superadmin); admin = project aktif
	Name             *string      `json:"name"`
	ReportType       *string      `json:"report_type"`
	ShiftCode        *string      `json:"shift_code"` // "" = satu hari penuh
	DateOffset       *int         `json:"date_offset"`
	SendTime         *string      `json:"send_time"`
	RecipientUserIDs *[]uuid.UUID `json:"recipient_user_ids"`
	IsActive         *bool        `json:"is_active"`
}

type ReportScheduleLog struct {
	ID           int64     `json:"id"`
	RunDate      string    `json:"run_date"`
	ReportDate   string    `json:"report_date"`
	Trigger      string    `json:"trigger"` // schedule | manual
	Status       string    `json:"status"`  // sent | failed
	Recipients   int       `json:"recipients"`
	ErrorMessage *string   `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}
