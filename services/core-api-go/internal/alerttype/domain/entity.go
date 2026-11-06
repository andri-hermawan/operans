// Package domain (fitur alerttype) = jenis alert (Overspeed, Fuel Decrease, …) + nilai bawaannya,
// serta aturan pengisian ambang yang juga dipakai fitur alertrule.
package domain

import (
	"math"

	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// AlertType = jenis alert + nilai bawaan (tabel core.alert_types, ADR 0008).
type AlertType struct {
	Code                   string   `json:"code"`
	LegacyCode             string   `json:"legacy_code"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	UsesDuration           bool     `json:"uses_duration"`
	UsesSpeedLimit         bool     `json:"uses_speed_limit"`
	UsesFuelDrop           bool     `json:"uses_fuel_drop"`
	DefaultDurationMinutes *float64 `json:"default_duration_minutes"`
	DefaultSpeedLimitKph   *float64 `json:"default_speed_limit_kph"`
	DefaultFuelDropLiters  *float64 `json:"default_fuel_drop_liters"`
}

// ApplyThresholds mengisi ambang; parameter yang tidak dipakai jenis alert ditolak.
// Dipakai saat mengubah nilai bawaan jenis alert (prefix "default_") dan saat menyimpan aturan alert (prefix "").
func ApplyThresholds(t *AlertType, dur, speed, fuel **float64, inDur, inSpeed, inFuel *float64, prefix string) error {
	check := func(dst **float64, v *float64, used bool, name string, max float64) error {
		if v == nil {
			return nil
		}
		if !used {
			return apperror.Invalid("%s%s tidak dipakai jenis alert %s", prefix, name, t.Name)
		}
		if *v < 0 || *v > max || math.IsNaN(*v) {
			return apperror.Invalid("%s%s harus 0..%v", prefix, name, max)
		}
		r := validate.Round2(*v)
		*dst = &r
		return nil
	}
	if err := check(dur, inDur, t.UsesDuration, "duration_minutes", 1440); err != nil {
		return err
	}
	if err := check(speed, inSpeed, t.UsesSpeedLimit, "speed_limit_kph", 300); err != nil {
		return err
	}
	return check(fuel, inFuel, t.UsesFuelDrop, "fuel_drop_liters", 1000)
}
