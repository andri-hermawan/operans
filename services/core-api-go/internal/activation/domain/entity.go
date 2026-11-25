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

// Activation = status operasi satu unit pada satu tanggal operasional. Kode unit disimpan sebagai teks (seperti tiket
// timbangan); EquipmentID terisi hanya bila kodenya ada di master equipment project itu.
type Activation struct {
	ID             int64      `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	EquipmentID    *uuid.UUID `json:"equipment_id"`
	EquipmentCode  string     `json:"equipment_code"`
	ActivationDate string     `json:"activation_date"` // YYYY-MM-DD
	Status         string     `json:"status"`
	Description    *string    `json:"description"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ActivationFilter = filter daftar aktivasi. Tanggal format YYYY-MM-DD.
type ActivationFilter struct {
	ProjectID     *uuid.UUID
	EquipmentCode string
	DateFrom      string
	DateTo        string
	Status        string
}

// ActivationRow = satu aktivasi yang sudah divalidasi (dari form maupun impor Excel).
type ActivationRow struct {
	ProjectID     uuid.UUID
	EquipmentCode string
	EquipmentID   *uuid.UUID
	Date          string
	Status        string
	Description   *string
}
