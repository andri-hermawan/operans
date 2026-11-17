import { useMemo } from 'react'

import { useEquipmentList } from './useEquipments'

/** id unit → tipe unit (Dump Truck, Water Truck, Fuel Truck, …); kosong bila belum diisi. */
export function useTypeOf() {
  const { data } = useEquipmentList(true)
  return useMemo(() => {
    const m = new Map((data ?? []).map((e) => [e.id, e.equipment_type]))
    return (id?: string | null) => (id && m.get(id)) || null
  }, [data])
}

/** id unit → kode unit. */
export function useCodeOf() {
  const { data } = useEquipmentList(true)
  return useMemo(() => {
    const m = new Map((data ?? []).map((e) => [e.id, e.equipment_code]))
    return (id?: string | null) => (id && m.get(id)) || '-'
  }, [data])
}
