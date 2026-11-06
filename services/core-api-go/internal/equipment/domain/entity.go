// Package domain (fitur equipment) = unit (dump truck, excavator, …) milik project.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Equipment = unit (dump truck, excavator, …) — tabel core.equipment.
type Equipment struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	EquipmentCode  string    `json:"equipment_code"`
	EquipmentAlias *string   `json:"equipment_alias"`
	EquipmentType  *string   `json:"equipment_type"`
	Brand          *string   `json:"brand"`
	Model          *string   `json:"model"`
	Class          *string   `json:"class"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EquipmentInput = isian tambah/ubah equipment. Pada ubah, nil = tidak diubah.
type EquipmentInput struct {
	ProjectID      *uuid.UUID `json:"project_id"` // hanya saat tambah
	EquipmentCode  *string    `json:"equipment_code"`
	EquipmentAlias *string    `json:"equipment_alias"`
	EquipmentType  *string    `json:"equipment_type"`
	Brand          *string    `json:"brand"`
	Model          *string    `json:"model"`
	Class          *string    `json:"class"`
	IsActive       *bool      `json:"is_active"`
}
