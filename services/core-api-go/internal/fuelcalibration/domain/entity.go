// Package domain (fitur fuelcalibration) = kalibrasi tangki BBM: nilai sensor → liter (ADR 0008).
package domain

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Jenis kalibrasi fuel (ADR 0008).
const (
	CalibrationLLS        = "lls"        // nilai mentah sensor level (mis. Escort 0–4095)
	CalibrationPercentage = "percentage" // persen dari ECU (0–100)
)

// FuelCalibration = kalibrasi tangki satu equipment (tabel core.fuel_calibrations + titiknya).
type FuelCalibration struct {
	EquipmentID        uuid.UUID          `json:"equipment_id"`
	EquipmentCode      string             `json:"equipment_code"`
	ProjectID          uuid.UUID          `json:"project_id"`
	CalibrationType    string             `json:"calibration_type"`
	TankCapacityLiters float64            `json:"tank_capacity_liters"`
	MaxSensorValue     float64            `json:"max_sensor_value"`
	IsLinear           bool               `json:"is_linear"`
	PointCount         int                `json:"point_count"`
	UpdatedAt          time.Time          `json:"updated_at"`
	Points             []CalibrationPoint `json:"points,omitempty"`
}

// CalibrationPoint = satu titik: nilai sensor → liter.
type CalibrationPoint struct {
	SensorValue  float64 `json:"sensor_value"`
	VolumeLiters float64 `json:"volume_liters"`
}

// LinearPoints membangkitkan titik tiap 5 liter dari kapasitas & nilai sensor maksimum
// (sama persis dengan sistem lama): LLS dibulatkan ke bilangan bulat, persen 2 desimal.
func LinearPoints(capacityLiters, maxSensor float64, calibrationType string) []CalibrationPoint {
	level := func(v float64) float64 {
		l := v / capacityLiters * maxSensor
		if calibrationType == CalibrationPercentage {
			return math.Round(l*100) / 100
		}
		return math.Round(l)
	}
	var pts []CalibrationPoint
	for v := 0.0; v <= capacityLiters; v += 5 {
		pts = append(pts, CalibrationPoint{SensorValue: level(v), VolumeLiters: v})
	}
	if last := pts[len(pts)-1]; last.VolumeLiters != capacityLiters {
		pts = append(pts, CalibrationPoint{SensorValue: maxSensor, VolumeLiters: capacityLiters})
	}
	return pts
}

// Liters = liter untuk nilai sensor dengan interpolasi linear di antara dua titik terdekat.
// ok=false bila nilai di luar rentang titik (sama dengan lama: tidak dihitung). points urut naik.
func Liters(points []CalibrationPoint, value float64) (float64, bool) {
	n := len(points)
	if n == 0 || math.IsNaN(value) || value < points[0].SensorValue || value > points[n-1].SensorValue {
		return 0, false
	}
	i := sort.Search(n, func(i int) bool { return points[i].SensorValue >= value }) // titik atas
	hi := points[i]
	if hi.SensorValue == value || i == 0 {
		return hi.VolumeLiters, true
	}
	lo := points[i-1]
	return lo.VolumeLiters + (value-lo.SensorValue)*(hi.VolumeLiters-lo.VolumeLiters)/(hi.SensorValue-lo.SensorValue), true
}
