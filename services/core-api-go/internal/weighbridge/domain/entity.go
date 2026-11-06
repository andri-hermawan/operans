// Package domain (fitur weighbridge) = tiket timbangan (berat kg; net = gross − tare).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// WeighbridgeTicket = satu tiket timbangan (tabel core.weighbridge_tickets; lama: weighbridge). Berat kg.
type WeighbridgeTicket struct {
	ID              int64      `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	OperationalDate string     `json:"operational_date"`
	ShiftID         uuid.UUID  `json:"shift_id"`
	ShiftCode       string     `json:"shift_code"`
	ShiftName       string     `json:"shift_name"`
	TicketNo        string     `json:"ticket_no"`
	EquipmentCode   string     `json:"equipment_code"`
	EquipmentID     *uuid.UUID `json:"equipment_id"`
	Product         *string    `json:"product"`
	GrossKg         float64    `json:"gross_kg"`
	TareKg          float64    `json:"tare_kg"`
	NetKg           float64    `json:"net_kg"`
	Recipient       *string    `json:"recipient"`
	Customer        *string    `json:"customer"`
	Transporter     *string    `json:"transporter"`
	GrossAt         *time.Time `json:"gross_at"`
	TareAt          *time.Time `json:"tare_at"`
	GrossOperator   *string    `json:"gross_operator"`
	TareOperator    *string    `json:"tare_operator"`
	Description     *string    `json:"description"`
	Location        *string    `json:"location"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
