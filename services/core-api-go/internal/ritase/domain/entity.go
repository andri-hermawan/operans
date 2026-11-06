// Package domain (fitur ritase) = Ritase: hasil angkut per jam kerja (loader + hauler, lokasi, material, produksi).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Ritase = satu baris Ritase (tabel core.ritases). Angka produksi disimpan apa adanya dari input.
type Ritase struct {
	ID                 int64      `json:"id"`
	ProjectID          uuid.UUID  `json:"project_id"`
	OperationalDate    string     `json:"operational_date"`
	ShiftID            uuid.UUID  `json:"shift_id"`
	ShiftCode          string     `json:"shift_code"`
	ShiftName          string     `json:"shift_name"`
	TimeSlot           string     `json:"time_slot"` // mis. 07.00-08.00
	LoaderCode         string     `json:"loader_code"`
	LoaderEquipmentID  *uuid.UUID `json:"loader_equipment_id"`
	LoaderModel        *string    `json:"loader_model"`
	LoaderOperatorNik  *string    `json:"loader_operator_nik"`
	LoaderOperatorName *string    `json:"loader_operator_name"`
	LoaderPartner      *string    `json:"loader_partner"`
	HaulerCode         string     `json:"hauler_code"`
	HaulerEquipmentID  *uuid.UUID `json:"hauler_equipment_id"`
	HaulerModel        *string    `json:"hauler_model"`
	HaulerOperatorNik  *string    `json:"hauler_operator_nik"`
	HaulerOperatorName *string    `json:"hauler_operator_name"`
	HaulerPartner      *string    `json:"hauler_partner"`
	Block              *string    `json:"block"`
	Location           *string    `json:"location"`
	Dumping            *string    `json:"dumping"`
	Material           *string    `json:"material"`
	SubMaterial        *string    `json:"sub_material"`
	Seam               *string    `json:"seam"`
	Phase              *string    `json:"phase"`
	Distance           *float64   `json:"distance"`
	Ritase             *float64   `json:"ritase"`
	Production         *float64   `json:"production"`
	ProductionAdjusted *float64   `json:"production_adjusted"`
	DistanceVolume     *float64   `json:"distance_volume"`
	ProductivityFactor *float64   `json:"productivity_factor"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
