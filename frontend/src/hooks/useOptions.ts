// Sumber pilihan untuk isian select. Halaman cukup menyebut sumbernya, bukan mengambil data sendiri.
import { useQuery } from '@tanstack/react-query'

import { useAuthStore } from '@/stores/auth.store'
import type { Option, OptionSource } from '@/types/form.types'

import { useEquipmentList } from './useEquipments'
import { useReferenceValues } from './useReferenceValues'
import { useShifts } from './useShifts'

/** Ubah sumber menjadi daftar pilihan. Semua hook selalu dipanggil (aturan React), datanya dipakai sesuai sumber. */
export function useOptions(source?: OptionSource): { options: Option[]; loading: boolean } {
  const kind = Array.isArray(source) ? 'static' : typeof source === 'string' ? source : !source ? 'none' : 'load' in source ? 'load' : 'reference'
  const shifts = useShifts()
  const equipment = useEquipmentList(false)
  const ref = useReferenceValues(kind === 'reference' ? (source as { reference: string }).reference : undefined)
  const pid = useAuthStore((s) => s.project?.project_id) ?? ''
  const loader = kind === 'load' ? (source as { key: string; load: (p: string) => Promise<Option[]> }) : undefined
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
    case 'reference':
      return { options: (ref.data ?? []).map((r) => ({ value: r.code, label: r.label })), loading: ref.isLoading }
    case 'load':
      return { options: loaded.data ?? [], loading: loaded.isLoading }
    default:
      return { options: [], loading: false }
  }
}
