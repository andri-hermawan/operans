// Bagian bersama 4 halaman Record / Upload (input harian core-api, ADR 0007).
import type { CrudConfig, CrudCtx } from '@/types/crud.types'
import type { FilterField, FilterValues } from '@/types/filter.types'
import type { FieldDef } from '@/types/form.types'
import { col } from '@/components/table/columns'

/** Filter standar input harian: tanggal operasional + shift (default: yang sedang berjalan). */
export const dailyFilters: FilterField[] = [{ type: 'date' }, { type: 'shift', allowAll: true }]
export const dailyDefaults: CrudConfig<unknown>['defaultFilters'] = (now) => ({ date: now.date })

/** Hubungkan fungsi API input harian (services/api/daily.api.ts) ke CrudPage: konteks project & shift diisi di sini. */
export const dailyCrud = <T extends { id: number }>(api: {
  list: (projectId: string, date?: string, shiftId?: string) => Promise<T[]>
  create: (body: Record<string, unknown>) => Promise<unknown>
  update: (id: number, body: Record<string, unknown>) => Promise<unknown>
  remove: (id: number) => Promise<unknown>
}) => ({
  list: (f: FilterValues, ctx: CrudCtx) => api.list(ctx.projectId, f.date, ctx.shiftId(f.shift)),
  create: (p: Record<string, unknown>, ctx: CrudCtx) => api.create({ ...p, project_id: ctx.projectId }),
  update: (row: T, p: Record<string, unknown>) => api.update(row.id, p),
  remove: (row: T) => api.remove(row.id),
})

/** Kolom tanggal + shift + unit yang ada di semua input harian. */
export const dateShiftUnitColumns = <T>() => [
  col.rowNo<T>(),
  col.plainDate<T>('operational_date', 'Date'),
  col.text<T>('shift_name', 'Shift'),
  col.code<T>('equipment_code', 'Asset ID', { mobile: 'title' }),
]

/** Isian tanggal + shift + unit (unit boleh diketik bila belum ada di master, seperti sistem lama). */
export const dateShiftUnitFields: FieldDef[] = [
  { name: 'operational_date', label: 'Tanggal', type: 'date', required: true },
  { name: 'shift', label: 'Shift', type: 'select', options: 'shifts', required: true },
  { name: 'equipment_code', label: 'Asset ID', type: 'combo', options: 'equipmentCodes', required: true },
]

/** Baris → nilai form untuk tanggal + shift. */
export const dailyToForm = (row: { operational_date: string; shift_code: string }) => ({
  ...row,
  shift: row.shift_code,
})
