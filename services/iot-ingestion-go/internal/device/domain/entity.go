// Package domain = bentuk data fitur Device (alat GPS di iot.devices) dan aturan murninya.
// Device juga dipakai fitur ingest sebagai "alat yang sedang mengirim data" (hasil login IMEI / API key).
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Device = alat yang terdaftar dan AKTIF di iot.devices.
type Device struct {
	ID          uuid.UUID
	IMEI        string
	EquipmentID *uuid.UUID // nil bila alat belum dipasang ke equipment
	ProjectID   *uuid.UUID // nil bila alat belum ditetapkan ke project
}

// DeviceRow = alat GPS untuk halaman System Data → Device (TANPA api_key_hash).
type DeviceRow struct {
	ID            uuid.UUID  `json:"id"`
	IMEI          string     `json:"imei"`
	Name          *string    `json:"name"`
	ProjectID     *uuid.UUID `json:"project_id"`
	EquipmentID   *uuid.UUID `json:"equipment_id"`
	EquipmentCode *string    `json:"equipment_code"`
	Model         *string    `json:"model"`
	FuelSource    *string    `json:"fuel_source"`
	SimNumber     *string    `json:"sim_number"`
	ProviderName  *string    `json:"provider_name"`
	HasAPIKey     bool       `json:"has_api_key"`
	IsActive      bool       `json:"is_active"`
	LastEventAt   *time.Time `json:"last_event_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DeviceInput = isian tambah/ubah; kolom nil = tidak diubah. "" pada teks = dikosongkan.
type DeviceInput struct {
	IMEI         *string    `json:"imei"`
	Name         *string    `json:"name"`
	ProjectID    *uuid.UUID `json:"project_id"`
	EquipmentID  *uuid.UUID `json:"equipment_id"`
	ClearUnit    bool       `json:"clear_equipment"` // true = lepas dari equipment
	Model        *string    `json:"model"`
	FuelSource   *string    `json:"fuel_source"`
	SimNumber    *string    `json:"sim_number"`
	ProviderName *string    `json:"provider_name"`
	IsActive     *bool      `json:"is_active"`
}

// ErrIMEIExists = IMEI sudah dipakai alat lain.
var ErrIMEIExists = errors.New("IMEI sudah terdaftar")

// HashKey = SHA-256 hex dari API key. Rumusnya sama dengan perintah PowerShell di Step 5g.
func HashKey(k string) string {
	sum := sha256.Sum256([]byte(k))
	return hex.EncodeToString(sum[:])
}
