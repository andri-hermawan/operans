import { useFilters } from './useFilters'
import { useOperationalDefaults } from './useOperationalDefaults'

interface Options {
  /**
   * true = tanggal & shift SELALU mengikuti shift yang sedang berjalan dan berganti otomatis saat shift/tanggal berganti
   * (parameter di alamat halaman diabaikan). Untuk halaman pemantauan live 24 jam yang kolom tanggal & shift-nya dikunci.
   */
  follow?: boolean
}

/** Filter tanggal operasional + shift (+ unit) di URL; awal = shift yang sedang berjalan. */
export function useSlotFilters(extra: Record<string, string | undefined> = {}, { follow = false }: Options = {}) {
  const now = useOperationalDefaults(follow)
  const [urlValues, set] = useFilters({ date: now.date, shift: now.shift, ...extra })
  const values = follow ? { ...urlValues, date: now.date, shift: now.shift } : urlValues
  return { values, set, ready: now.ready, slot: { date: values.date, shift: values.shift, equipmentId: values.equipment } }
}
