// Record / Upload → Plan → Plan Yearly (target per bulan dalam satu tahun, per mitra, lokasi, deskripsi, dan nama).
import { yearlyPlanApi } from '@/services/api/plan.api'
import type { CrudConfig } from '@/types/crud.types'
import type { FieldDef } from '@/types/form.types'
import type { YearlyPlan } from '@/types/plan.types'
import { MONTH_KEYS, MONTH_TITLES } from '@/utils/plan'
import CrudPage from '@/components/crud/CrudPage'
import { planPayload, planYearDefaults, yearFilter } from '@/components/crud/planConfig'
import { col } from '@/components/table/columns'

const yearlyConfig: CrudConfig<YearlyPlan> = {
  title: 'Plan Yearly',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'yearly-plans',
  filters: [yearFilter],
  defaultFilters: planYearDefaults,
  list: (f, ctx) => yearlyPlanApi.list(ctx.projectId, f.year),
  searchable: true,
  columns: [
    col.rowNo(),
    col.code('partner', 'Mitra', { mobile: 'title' }),
    col.text('plan_year', 'Year'),
    col.text('location', 'Location'),
    col.text('description', 'Description'),
    col.text('plan_name', 'Name'),
    col.number('total', 'Total', 2),
    ...MONTH_KEYS.map((k) => col.number<YearlyPlan>(k, MONTH_TITLES[k], 2, { mobile: 'hide' })),
  ],
  form: {
    width: 760,
    fields: [
      { name: 'partner', label: 'Mitra', required: true },
      { name: 'plan_year', label: 'Year', type: 'number', required: true, min: 2000, max: 2100 },
      { name: 'location', label: 'Location', required: true, placeholder: 'mis. PIT A2' },
      { name: 'description', label: 'Description', required: true, placeholder: 'mis. Volume' },
      { name: 'plan_name', label: 'Name', required: true, span: 'full', placeholder: 'mis. Waste Removal' },
      { name: 'total', label: 'Total', type: 'number', min: 0, span: 'full', help: 'Kosongkan agar dihitung otomatis dari jumlah bulan.' },
      ...MONTH_KEYS.map((k): FieldDef => ({ name: k, label: MONTH_TITLES[k], type: 'number', min: 0 })),
    ],
    initial: (_c, f) => ({ plan_year: Number(f.year) }),
    create: (p, ctx) => yearlyPlanApi.create({ ...planPayload(p), project_id: ctx.projectId }),
    update: (row, p) => yearlyPlanApi.update(row.id, planPayload(p)),
  },
  remove: { action: (row) => yearlyPlanApi.remove(row.id), label: (r) => `plan ${r.partner} ${r.plan_year} ${r.plan_name}` },
  importType: 'yearly-plans',
  exportName: 'plan_yearly',
}

export default function PlanYearlyPage() {
  return <CrudPage config={yearlyConfig} />
}
