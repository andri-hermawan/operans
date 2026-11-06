package domain

// Kode status operasi equipment (ADR 0004).
const (
	StatusRunning = "running"
	StatusIdle    = "idle"
	StatusStop    = "stop"
	StatusOffline = "offline"
)

// OperationalStatus = aturan status operasi sistem lama (ADR 0005, keputusan 1):
//
//	sinyal GSM ≤ 0 atau operator GSM kosong → offline
//	mesin hidup & speed > 0               → running
//	mesin hidup & speed 0                 → idle
//	mesin mati                            → stop
//
// Seperti sistem lama, nilai yang tidak dikirim alat dianggap 0 / mesin mati.
func OperationalStatus(e *Event) string {
	if e.GSMSignalLevel == nil || *e.GSMSignalLevel <= 0 ||
		e.GSMOperatorCode == nil || *e.GSMOperatorCode == 0 {
		return StatusOffline
	}
	if e.IsEngineOn == nil || !*e.IsEngineOn {
		return StatusStop
	}
	if e.SpeedKph != nil && *e.SpeedKph > 0 {
		return StatusRunning
	}
	return StatusIdle
}
