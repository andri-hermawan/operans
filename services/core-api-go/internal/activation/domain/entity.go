// Package domain (fitur activation) = status operasi harian equipment (aktif / tidak) per tanggal.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status aktivasi harian (ADR 0004; lama: Active / Non Active).
const (
	ActivationActive   = "active"
	ActivationInactive = "inactive"
)

// Activation = status operasi satu equipment pada satu tanggal operasional.
type Activation struct {
	ID             int64     `json:"id"`
	EquipmentID    uuid.UUID `json:"equipment_id"`
	EquipmentCode  string    `json:"equipment_code"`
	ProjectID      uuid.UUID `json:"project_id"`
	ActivationDate string    `json:"activation_date"` // YYYY-MM-DD
	Status         string    `json:"status"`
	Description    *string   `json:"description"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ActivationFilter = filter daftar aktivasi. Tanggal format YYYY-MM-DD.
type ActivationFilter struct {
	ProjectID   *uuid.UUID
	EquipmentID *uuid.UUID
	DateFrom    string
	DateTo      string
	Status      string
}

// ActivationRow = satu baris impor aktivasi yang sudah divalidasi.
type ActivationRow struct {
	EquipmentID uuid.UUID
	Date        string
	Status      string
	Description *string
}
