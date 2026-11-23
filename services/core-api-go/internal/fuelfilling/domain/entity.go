// Package domain (fitur fuelfilling) = Fuel Filling: pengisian BBM tiap unit per tanggal operasional + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// FuelFilling = satu baris pengisian BBM (tabel core.fuel_fillings). HM dalam jam, BBM dalam liter.
// TotalHours & FuelConsumption disimpan apa adanya dari input; kosong → dihitung.
type FuelFilling struct {
	ID              int64      `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	OperationalDate string     `json:"operational_date"`
	ShiftID         uuid.UUID  `json:"shift_id"`
	ShiftCode       string     `json:"shift_code"`
	ShiftName       string     `json:"shift_name"`
	EquipmentCode   string     `json:"equipment_code"`
	EquipmentID     *uuid.UUID `json:"equipment_id"`
	EquipmentModel  *string    `json:"equipment_model"`
	HmPrevious      *float64   `json:"hm_previous"`
	HmFilling       *float64   `json:"hm_filling"`
	TotalHours      *float64   `json:"total_hours"`
	FuelUsage       *float64   `json:"fuel_usage_liter"`
	FuelConsumption *float64   `json:"fuel_consumption_lph"`
	WasteRemoval    *float64   `json:"waste_removal_liter"`
	CoalMining      *float64   `json:"coal_mining_liter"`
	Hauling         *float64   `json:"hauling_liter"`
	Rental          *float64   `json:"rental_liter"`
	Dewatering      *float64   `json:"dewatering_liter"`
	FinalSupport    *float64   `json:"final_support_liter"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
