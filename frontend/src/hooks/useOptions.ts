// Sumber pilihan untuk isian select. Halaman cukup menyebut sumbernya, bukan mengambil data sendiri.
import { useQuery } from '@tanstack/react-query'

import { listDevices } from '@/services/api/device.api'
import { listFuelCalibrations } from '@/services/api/fuelCalibration.api'
import { useAuthStore } from '@/stores/auth.store'
import type { Option, OptionSource } from '@/types/form.types'

import { useEquipmentList } from './useEquipments'
import { useReferenceValues } from './useReferenceValues'
import { useShifts } from './useShifts'

/**
 * Ubah sumber menjadi daftar pilihan. Semua hook selalu dipanggil (aturan React), datanya dipakai sesuai sumber.
 * current = nilai yang sedang terpilih di isian; tetap ikut ditampilkan pada sumber yang menyaring unit terdaftar.
 */
export function useOptions(source?: OptionSource, current?: string): { options: Option[]; loading: boolean } {
  const kind = Array.isArray(source) ? 'static' : typeof source === 'string' ? source : !source ? 'none' : 'load' in source ? 'load' : 'reference'
  const shifts = useShifts()
  const equipment = useEquipmentList(false)
  const ref = useReferenceValues(kind === 'reference' ? (source as { reference: string }).reference : undefined)
  const pid = useAuthStore((s) => s.project?.project_id) ?? ''
  const loader = kind === 'load' ? (source as { key: string; load: (p: string) => Promise<Option[]> }) : undefined
  // unit yang sudah terdaftar (punya baris Device / kalibrasi fuel): hanya dimuat bila sumbernya memerlukan; selalu segar saat form dibuka. Terdaftar = sudah ada baris di tabel menunya sendiri.
  const registry = useQuery({
    queryKey: [pid, 'options', 'registered', kind],
    enabled: !!pid && (kind === 'equipmentWithoutDevice' || kind === 'equipmentWithoutCalibration'),
    staleTime: 0,
    refetchOnMount: 'always',
    queryFn: async (): Promise<string[]> =>
      kind === 'equipmentWithoutDevice'
        ? (await listDevices()).filter((d) => d.equipment_id).map((d) => d.equipment_id as string)
        : (await listFuelCalibrations(pid)).map((c) => c.equipment_id),
  })
  const loaded = useQuery({ queryKey: [pid, 'options', loader?.key], enabled: !!loader && !!pid, staleTime: 60_000, queryFn: () => loader!.load(pid) })

  switch (kind) {
    case 'static':
      return { options: source as Option[], loading: false }
    case 'shifts':
      return { options: (shifts.data ?? []).map((s) => ({ value: s.shift_code, label: s.shift_name })), loading: shifts.isLoading }
    case 'equipmentCodes':
      return { options: (equipment.data ?? []).map((e) => ({ value: e.equipment_code, label: e.equipment_code })), loading: equipment.isLoading }
    case 'equipmentIds':
      return { options: (equipment.data ?? []).map((e) => ({ value: e.id, label: e.equipment_code })), loading: equipment.isLoading }
    case 'equipmentWithoutDevice':
    case 'equipmentWithoutCalibration': {
      const taken = new Set(registry.data ?? [])
      const free = (equipment.data ?? []).filter((e) => !taken.has(e.id) || e.id === current)
      return { options: free.map((e) => ({ value: e.id, label: e.equipment_code })), loading: equipment.isLoading || registry.isLoading }
    }
    case 'reference':
      return { options: (ref.data ?? []).map((r) => ({ value: r.code, label: r.label })), loading: ref.isLoading }
    case 'load':
      return { options: loaded.data ?? [], loading: loaded.isLoading }
    default:
      return { options: [], loading: false }
  }
}
