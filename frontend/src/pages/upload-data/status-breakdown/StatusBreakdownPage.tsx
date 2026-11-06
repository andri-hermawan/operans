// Record / Upload → Status Breakdown.
import { Tag } from 'antd'

import { breakdownApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { EquipmentBreakdown } from '@/types/daily.types'
import { hhmm } from '@/utils/time'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitColumns, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const BREAKDOWN_STATUS = [
  { value: 'breakdown', label: 'Breakdown' },
  { value: 'continue', label: 'Continue' },
  { value: 'ready', label: 'Ready' },
]
const breakdownCrud = dailyCrud(breakdownApi)
const breakdownConfig: CrudConfig<EquipmentBreakdown> = {
  title: 'Status Breakdown',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'equipment-breakdowns',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: breakdownCrud.list,
  columns: [
    ...dateShiftUnitColumns<EquipmentBreakdown>(),
    { key: 'status', title: 'Status', render: (r) => <Tag color={r.status === 'ready' ? 'success' : r.status === 'continue' ? 'warning' : 'error'}>{BREAKDOWN_STATUS.find((s) => s.value === r.status)?.label ?? r.status}</Tag>, value: (r) => r.status },
    col.text('category_label', 'Category'),
    col.clock('started_at', 'Start'),
    col.clock('ended_at', 'End'),
    col.number('duration_hours', 'Duration (jam)', 2),
    { key: 'repair_status_label', title: 'Repair Status', render: (r) => <Tag color={r.repair_status_code === 'done' ? 'success' : 'orange'}>{r.repair_status_label}</Tag> },
    col.text('description', 'Description'),
    col.text('location', 'Location'),
  ],
  form: {
    width: 640,
    fields: [
      ...dateShiftUnitFields,
      { name: 'status', label: 'Status', type: 'select', options: BREAKDOWN_STATUS, required: true, initial: 'breakdown' },
      { name: 'category', label: 'Category', type: 'select', options: { reference: 'breakdown_category' }, required: true },
      { name: 'repair_status', label: 'Repair Status', type: 'select', options: { reference: 'repair_status' }, required: true },
      { name: 'time_start', label: 'Jam Mulai', type: 'time', required: true },
      { name: 'time_end', label: 'Jam Selesai', type: 'time', help: 'Kosong pada status Continue = sampai akhir shift. Durasi dihitung otomatis.' },
      { name: 'location', label: 'Location' },
      { name: 'description', label: 'Description', type: 'textarea' },
    ],
    toForm: (r) => ({ ...dailyToForm(r), category: r.category_code, repair_status: r.repair_status_code, time_start: hhmm(r.started_at), time_end: hhmm(r.ended_at) }),
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: breakdownCrud.create,
    update: breakdownCrud.update,
  },
  remove: { action: breakdownCrud.remove, label: (r) => `breakdown ${r.equipment_code}` },
  importType: 'equipment-breakdowns',
  exportName: 'status_breakdown',
}

export default function StatusBreakdownPage() {
  return <CrudPage config={breakdownConfig} />
}
