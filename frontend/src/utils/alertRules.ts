// Ambang yang dipakai tiap jenis alert (sama dengan core.alert_types), untuk menampilkan isian yang relevan.

export const ALERT_USES: Record<string, { duration: boolean; speed: boolean; fuel: boolean }> = {
  overspeed: { duration: true, speed: true, fuel: false },
  underspeed: { duration: true, speed: true, fuel: false },
  off_track: { duration: true, speed: false, fuel: false },
  fuel_decrease_engine_on: { duration: true, speed: false, fuel: true },
  fuel_decrease_engine_off: { duration: false, speed: false, fuel: true },
}
export const uses = (code: unknown, k: 'duration' | 'speed' | 'fuel') => !code || ALERT_USES[String(code)]?.[k] !== false
