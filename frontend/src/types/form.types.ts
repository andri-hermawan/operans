// Definisi isian form (dipakai FormFields & CrudPage) dan sumber pilihan select.
import type { ReactNode } from 'react'
import type { Rule } from 'antd/es/form'

import type { Role } from './auth.types'

export interface Option {
  value: string
  label: string
}

export type OptionSource =
  | Option[]
  | 'shifts' // value = shift_code
  | 'equipmentCodes' // value = equipment_code (unit aktif)
  | 'equipmentIds' // value = id
  | { reference: string } // core.reference_values group (breakdown_category, repair_status)
  | { key: string; load: (projectId: string) => Promise<Option[]> } // sumber lain milik halaman (di-cache per project)

export type FieldType =
  | 'text' | 'upper' | 'email' | 'phone' | 'password' | 'textarea'
  | 'number' | 'date' | 'time' | 'switch' | 'radio' | 'select' | 'multiSelect' | 'combo'

export interface FieldDef {
  name: string
  label: string
  type?: FieldType
  required?: boolean
  rules?: Rule[]
  options?: OptionSource
  placeholder?: string
  /** Satuan di belakang angka (L, kg, km/h, menit). */
  unit?: string
  min?: number
  max?: number
  /** Lebar: 'full' = satu baris penuh. */
  span?: 'half' | 'full'
  /** Tampil hanya bila kondisi terpenuhi (nilai form saat ini, mode tambah/ubah). */
  visible?: (values: Record<string, unknown>, mode: 'create' | 'edit') => boolean
  disabled?: (values: Record<string, unknown>, mode: 'create' | 'edit') => boolean
  help?: ReactNode
  /** Nilai awal saat tambah. */
  initial?: unknown
  /** Hanya tampil untuk role ini (mis. status Super Admin hanya diatur superadmin). */
  roles?: Role[]
}
