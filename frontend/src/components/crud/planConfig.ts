// Bagian bersama halaman Plan Yearly & Plan Monthly (Record / Upload → Plan).
import type { CrudConfig, Payload } from '@/types/crud.types'
import type { FilterField, FilterValues } from '@/types/filter.types'
import { MONTH_NAMES, yearOptions } from '@/utils/plan'

/** Filter tahun (selalu terisi; bawaan tahun operasional berjalan). */
export const yearFilter: FilterField = { type: 'select', key: 'year', label: 'Tahun', options: yearOptions(new Date().getFullYear()) }

/** Filter bulan (boleh "Semua": menampilkan semua bulan pada tahun itu). */
export const monthFilter: FilterField = {
  type: 'select',
  key: 'month',
  label: 'Bulan',
  allowAll: true,
  options: MONTH_NAMES.map((name, i) => ({ value: String(i + 1), label: name })),
}

export const planYearDefaults: CrudConfig<unknown>['defaultFilters'] = (now): FilterValues => ({ year: now.date.slice(0, 4) })

export const planMonthDefaults: CrudConfig<unknown>['defaultFilters'] = (now): FilterValues => ({
  year: now.date.slice(0, 4),
  month: String(Number(now.date.slice(5, 7))),
})

/** Pilihan bulan di form berupa teks ("6"); backend menerima angka. */
export const planPayload = (p: Payload): Payload => ({
  ...p,
  ...(p.plan_year != null && p.plan_year !== '' && { plan_year: Number(p.plan_year) }),
  ...(p.plan_month != null && p.plan_month !== '' && { plan_month: Number(p.plan_month) }),
})
