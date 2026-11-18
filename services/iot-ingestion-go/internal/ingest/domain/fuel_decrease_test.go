package domain

import (
	"math/rand"
	"testing"
	"time"
)

// bacaan uji FUD: detik ke-n sejak 08:00:00, speed, mesin, liter (nil = bacaan tidak sah)
type fudStep struct {
	sec    int
	speed  float64
	engine bool
	liters float64
}

// runFUD menjalankan DetectFuel dengan aturan bawaan (5 L / 2 menit) dan mengumpulkan kejadian FUD saja.
func runFUD(steps []fudStep) []FuelOutcome {
	t0 := time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)
	var tr Tracker
	var all []FuelOutcome
	for _, s := range steps {
		s := s
		e := &Event{EventTime: t0.Add(time.Duration(s.sec) * time.Second)}
		e.SpeedKph, e.IsEngineOn = &s.speed, &s.engine
		l := s.liters
		var out []FuelOutcome
		tr, out = DetectFuel(tr, e, &l, DefaultAlertRules)
		for _, o := range out {
			if o.AlertType == AlertFuelDecreaseEngineOn {
				all = append(all, o)
			}
		}
	}
	return all
}

// calm = unit berhenti, bacaan tiap 10 detik dari detik `from` sampai `to` dengan derau kecil di sekitar level(t).
func calm(from, to int, level func(sec int) float64, noise float64, r *rand.Rand) []fudStep {
	var out []fudStep
	for s := from; s <= to; s += 10 {
		out = append(out, fudStep{s, 0, true, level(s) + (r.Float64()*2-1)*noise})
	}
	return out
}

func flat(v float64) func(int) float64 { return func(int) float64 { return v } }

func TestFUD(t *testing.T) {
	r := rand.New(rand.NewSource(7))

	t.Run("penyedotan searah setelah tenang → alert dengan total turun", func(t *testing.T) {
		steps := calm(0, 200, flat(120), 0.1, r)
		// mulai detik 210 turun 3 L/menit
		steps = append(steps, calm(210, 480, func(s int) float64 { return 120 - float64(s-200)*3.0/60 }, 0.1, r)...)
		out := runFUD(steps)
		if len(out) < 2 || out[0].Extend || out[0].EventType != "decrease" || out[0].DeltaLiter > -5 {
			t.Fatalf("seharusnya alert baru lalu perpanjangan: %+v", out)
		}
		last := out[len(out)-1]
		if !last.Extend || last.DeltaLiter > out[0].DeltaLiter {
			t.Fatalf("total turun harus bertambah saat diperpanjang: %+v", out)
		}
		for _, o := range out[1:] {
			if !o.StartedAt.Equal(out[0].StartedAt) {
				t.Fatalf("perpanjangan harus memakai awal yang sama: %+v", out)
			}
		}
	})

	t.Run("idle lama: turun pelan 3 L dalam 1 jam → tanpa alert", func(t *testing.T) {
		out := runFUD(calm(0, 3600, func(s int) float64 { return 120 - float64(s)*3.0/3600 }, 0.1, r))
		if len(out) != 0 {
			t.Fatalf("idle normal tidak boleh jadi alert: %+v", out)
		}
	})

	t.Run("goyangan setelah berhenti (cairan belum tenang) → tanpa alert", func(t *testing.T) {
		var steps []fudStep
		for s := 0; s <= 120; s += 10 { // 2 menit pertama: goyang ±8 L
			steps = append(steps, fudStep{s, 0, true, 120 + (r.Float64()*2-1)*8})
		}
		steps = append(steps, calm(130, 600, flat(118), 0.1, r)...) // lalu tenang di level yang sama
		if out := runFUD(steps); len(out) != 0 {
			t.Fatalf("goyangan tidak boleh jadi alert: %+v", out)
		}
	})

	t.Run("sensor naik-turun sesudah waktu tenang → jendela di-reset, tanpa alert", func(t *testing.T) {
		steps := calm(0, 200, flat(120), 0.1, r)
		// koreksi sensor: naik-turun ±5 L selama ~3 menit, rata-rata tetap 120 → median tidak turun 5 L
		for s := 210; s <= 400; s += 10 {
			d := 5.0
			if (s/10)%2 == 0 {
				d = -5.0
			}
			steps = append(steps, fudStep{s, 0, true, 120 + d})
		}
		steps = append(steps, calm(410, 700, flat(120), 0.1, r)...)
		if out := runFUD(steps); len(out) != 0 {
			t.Fatalf("koreksi sensor tidak boleh jadi alert: %+v", out)
		}
	})

	t.Run("speed > 0 mereset: turun saat bergerak lalu berhenti lagi → tanpa alert", func(t *testing.T) {
		steps := calm(0, 200, flat(120), 0.1, r)
		for s := 210; s <= 300; s += 10 {
			steps = append(steps, fudStep{s, 25, true, 100}) // bergerak, level 100
		}
		steps = append(steps, calm(310, 700, flat(100), 0.1, r)...)
		if out := runFUD(steps); len(out) != 0 {
			t.Fatalf("%+v", out)
		}
	})

	t.Run("celah data > 60 detik mereset: lompatan sesudah sensor pulih tidak dihitung", func(t *testing.T) {
		steps := calm(0, 200, flat(120), 0.1, r)
		steps = append(steps, calm(400, 800, flat(110), 0.1, r)...) // celah 200 dtk, level 120 → 110
		if out := runFUD(steps); len(out) != 0 {
			t.Fatalf("lompatan setelah celah tidak boleh jadi alert: %+v", out)
		}
	})

	t.Run("mesin mati mereset: FUD hanya untuk mesin hidup", func(t *testing.T) {
		var steps []fudStep
		for s := 0; s <= 600; s += 30 {
			steps = append(steps, fudStep{s, 0, false, 120 - float64(s)*0.05}) // mesin mati, turun 30 L
		}
		if out := runFUD(steps); len(out) != 0 {
			t.Fatalf("%+v", out)
		}
	})

	t.Run("penjaga lambat: turun 12 L dalam 15 menit setelah tenang → alert", func(t *testing.T) {
		// tenang 0..600 di 120 L; mulai detik 700 turun 12 L dalam 15 menit (0.8 L/menit, di bawah ambang 5 L/2 mnt)
		steps := calm(0, 700, func(s int) float64 {
			if s < 700 {
				return 120
			}
			return 120
		}, 0.1, r)
		steps = append(steps, calm(710, 2400, func(s int) float64 {
			d := float64(s-700) * 12.0 / 900
			if d > 12 {
				d = 12
			}
			return 120 - d
		}, 0.1, r)...)
		out := runFUD(steps)
		if len(out) == 0 || out[0].EventType != "decrease" {
			t.Fatalf("penurunan lambat besar seharusnya terdeteksi: %+v", out)
		}
	})

	t.Run("satu kejadian = satu alert; kejadian terpisah = alert terpisah", func(t *testing.T) {
		steps := calm(0, 200, flat(150), 0.1, r)
		steps = append(steps, calm(210, 420, func(s int) float64 { return 150 - float64(s-200)*4.0/60 }, 0.1, r)...) // turun 14 L
		steps = append(steps, calm(430, 1400, flat(136), 0.1, r)...)                                                 // stabil lama
		steps = append(steps, calm(1410, 1620, func(s int) float64 { return 136 - float64(s-1400)*4.0/60 }, 0.1, r)...)
		steps = append(steps, calm(1630, 2000, flat(122), 0.1, r)...)
		out := runFUD(steps)
		opens := 0
		for _, o := range out {
			if !o.Extend {
				opens++
			}
		}
		if opens != 2 {
			t.Fatalf("seharusnya 2 alert terpisah, dapat %d: %+v", opens, out)
		}
	})
}
