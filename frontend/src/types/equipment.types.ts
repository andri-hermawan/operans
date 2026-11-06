// Unit / alat (data induk core-api).

export interface Equipment {
  id: string
  project_id: string
  equipment_code: string
  equipment_alias: string | null
  equipment_type: string | null
  brand: string | null
  model: string | null
  class: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

/** Baris Equipment Graphic: unit + ketersediaan dari Status Breakdown. */
export type EquipmentAvailabilityRow = Equipment & { availability: 'Ready' | 'Breakdown' | 'Continue' }
