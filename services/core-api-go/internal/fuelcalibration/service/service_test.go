package service

import (
	"testing"

	"simha/core-api/internal/fuelcalibration/domain"
)

func f64(v float64) *float64 { return &v }

func pt(sensor, liters float64) domain.CalibrationPoint {
	return domain.CalibrationPoint{SensorValue: sensor, VolumeLiters: liters}
}

func TestBuildCalibration(t *testing.T) {
	// isian lama: kapasitas + nilai maksimum → titik tiap 5 liter
	c, err := buildCalibration(FuelCalibrationInput{CalibrationType: "LLS", TankCapacityLiters: f64(209), MaxSensorValue: f64(4095)})
	if err != nil || !c.IsLinear || len(c.Points) != 43 || c.CalibrationType != domain.CalibrationLLS {
		t.Fatalf("linear: %+v err=%v", c, err)
	}
	// ECU persen: max otomatis 100
	c, err = buildCalibration(FuelCalibrationInput{CalibrationType: "percentage", TankCapacityLiters: f64(300)})
	if err != nil || c.MaxSensorValue != 100 {
		t.Fatalf("persen: %+v err=%v", c, err)
	}
	// tabel uji isi tangki (tidak urut di isian → diurutkan), kapasitas & max dari titik terakhir
	c, err = buildCalibration(FuelCalibrationInput{Points: []domain.CalibrationPoint{pt(4000, 200), pt(0, 0), pt(2000, 120)}})
	if err != nil || c.IsLinear || c.TankCapacityLiters != 200 || c.MaxSensorValue != 4000 || c.Points[1].SensorValue != 2000 {
		t.Fatalf("tabel: %+v err=%v", c, err)
	}

	for name, in := range map[string]FuelCalibrationInput{
		"jenis salah":              {CalibrationType: "volt", TankCapacityLiters: f64(200), MaxSensorValue: f64(4095)},
		"kapasitas kosong":         {MaxSensorValue: f64(4095)},
		"max lls kosong":           {TankCapacityLiters: f64(200)},
		"persen max bukan 100":     {CalibrationType: "percentage", TankCapacityLiters: f64(200), MaxSensorValue: f64(4095)},
		"titik cuma satu":          {Points: []domain.CalibrationPoint{pt(0, 0)}},
		"sensor dobel":             {Points: []domain.CalibrationPoint{pt(0, 0), pt(100, 10), pt(100, 12)}},
		"liter turun":              {Points: []domain.CalibrationPoint{pt(0, 0), pt(100, 20), pt(200, 10)}},
		"kapasitas beda dgn titik": {TankCapacityLiters: f64(250), Points: []domain.CalibrationPoint{pt(0, 0), pt(100, 20)}},
		"persen di atas 100":       {CalibrationType: "percentage", Points: []domain.CalibrationPoint{pt(0, 0), pt(120, 20)}},
	} {
		if _, err := buildCalibration(in); err == nil {
			t.Errorf("%s: seharusnya ditolak", name)
		}
	}
}
