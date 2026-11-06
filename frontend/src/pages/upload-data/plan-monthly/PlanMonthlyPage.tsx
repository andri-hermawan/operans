// Record / Upload → Plan → Plan Monthly (target per tanggal dalam satu bulan, per mitra, deskripsi, dan nama).
import { monthlyPlanApi } from '@/services/api/plan.api'
import type { CrudConfig } from '@/types/crud.types'
import type { FieldDef } from '@/types/form.types'
import type { MonthlyPlan } from '@/types/plan.types'
import { DAY_KEYS, MONTH_NAMES, dayTitle } from '@/utils/plan'
import CrudPage from '@/components/crud/CrudPage'
import { monthFilter, planMonthDefaults, planPayload, yearFilter } from '@/components/crud/planConfig'
import { col } from '@/components/table/columns'

const monthlyConfig: CrudConfig<MonthlyPlan> = {
  title: 'Plan Monthly',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'monthly-plans',
  filters: [yearFilter, monthFilter],
  defaultFilters: planMonthDefaults,
  list: (f, ctx) => monthlyPlanApi.list(ctx.projectId, f.year, f.month),
  searchable: true,
  subtitle: 'Tanggal yang tidak ada pada bulan itu (mis. 31 Juni) harus 0.',
  columns: [
    col.rowNo(),
    col.code('partner', 'Mitra', { mobile: 'title' }),
    col.text('plan_year', 'Year'),
    col.text('plan_month', 'Month', { render: (r) => MONTH_NAMES[r.plan_month - 1] ?? r.plan_month, value: (r) => r.plan_month }),
    col.text('description', 'Description'),
    col.text('plan_name', 'Name'),
    col.text('uom', 'Uom'),
    col.number('total', 'Total', 2),
    ...DAY_KEYS.map((k) => col.number<MonthlyPlan>(k, dayTitle(k), 2, { mobile: 'hide' })),
  ],
  form: {
    width: 760,
    fields: [
      { name: 'partner', label: 'Mitra', required: true },
      { name: 'plan_year', label: 'Year', type: 'number', required: true, min: 2000, max: 2100 },
      {
        name: 'plan_month',
        label: 'Month',
        type: 'select',
        required: true,
        options: MONTH_NAMES.map((n, i) => ({ value: String(i + 1), label: n })),
      },
      { name: 'uom', label: 'Uom', required: true, placeholder: 'mis. BCM' },
      { name: 'description', label: 'Description', required: true, placeholder: 'mis. Volume' },
      { name: 'plan_name', label: 'Name', required: true, placeholder: 'mis. Waste Removal' },
      { name: 'total', label: 'Total', type: 'number', min: 0, span: 'full', help: 'Kosongkan agar dihitung otomatis dari jumlah tanggal.' },
      ...DAY_KEYS.map((k): FieldDef => ({ name: k, label: dayTitle(k), type: 'number', min: 0 })),
    ],
    toForm: (row) => ({ ...row, plan_month: String(row.plan_month) }),
    initial: (_c, f) => ({ plan_year: Number(f.year), plan_month: f.month }),
    create: (p, ctx) => monthlyPlanApi.create({ ...planPayload(p), project_id: ctx.projectId }),
    update: (row, p) => monthlyPlanApi.update(row.id, planPayload(p)),
  },
  remove: {
    action: (row) => monthlyPlanApi.remove(row.id),
    label: (r) => `plan ${r.partner} ${MONTH_NAMES[r.plan_month - 1] ?? r.plan_month} ${r.plan_year} ${r.plan_name}`,
  },
  importType: 'monthly-plans',
  exportName: 'plan_monthly',
}

export default function PlanMonthlyPage() {
  return <CrudPage config={monthlyConfig} />
}
