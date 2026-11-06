// Package domain (fitur alertrule) = aturan alert per project / zona, penerima, dan ambang yang berlaku.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AlertRule = aturan alert project (zona NULL) atau zona. Ambang nil = ikut lapis di atasnya.
type AlertRule struct {
	ID               uuid.UUID        `json:"id"`
	ProjectID        uuid.UUID        `json:"project_id"`
	AlertType        string           `json:"alert_type"`
	AlertTypeName    string           `json:"alert_type_name"`
	ZoneName         *string          `json:"zone_name"`
	DurationMinutes  *float64         `json:"duration_minutes"`
	SpeedLimitKph    *float64         `json:"speed_limit_kph"`
	FuelDropLiters   *float64         `json:"fuel_drop_liters"`
	Description      *string          `json:"description"`
	IsActive         bool             `json:"is_active"`
	Recipients       []AlertRecipient `json:"recipients"`
	RecipientUserIDs []uuid.UUID      `json:"-"` // isian simpan
	UpdatedAt        time.Time        `json:"updated_at"`
}

// AlertRecipient = penerima notifikasi (WhatsApp ke phone_number).
type AlertRecipient struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	PhoneNumber *string   `json:"phone_number"`
}

// EffectiveAlertRule = ambang yang benar-benar berlaku untuk satu project + zona.
type EffectiveAlertRule struct {
	AlertType       string           `json:"alert_type"`
	Name            string           `json:"name"`
	DurationMinutes *float64         `json:"duration_minutes"`
	SpeedLimitKph   *float64         `json:"speed_limit_kph"`
	FuelDropLiters  *float64         `json:"fuel_drop_liters"`
	Recipients      []AlertRecipient `json:"recipients"`
}
