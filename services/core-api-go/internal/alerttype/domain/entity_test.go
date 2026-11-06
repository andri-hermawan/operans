package domain

import "testing"

func f64(v float64) *float64 { return &v }

func TestApplyThresholds(t *testing.T) {
	fuelOff := &AlertType{Name: "Fuel Decrease Engine Off", UsesFuelDrop: true}
	var dur, speed, fuel *float64
	if err := ApplyThresholds(fuelOff, &dur, &speed, &fuel, nil, nil, f64(7.456), ""); err != nil || *fuel != 7.46 {
		t.Fatalf("fuel: %v %v", err, fuel)
	}
	if err := ApplyThresholds(fuelOff, &dur, &speed, &fuel, nil, f64(50), nil, ""); err == nil {
		t.Fatal("speed limit untuk alert fuel seharusnya ditolak")
	}
	overspeed := &AlertType{Name: "Overspeed", UsesDuration: true, UsesSpeedLimit: true}
	if err := ApplyThresholds(overspeed, &dur, &speed, &fuel, nil, f64(400), nil, ""); err == nil {
		t.Fatal("speed 400 km/j seharusnya ditolak")
	}
}
