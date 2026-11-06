// Record / Upload → Transaction → WH & Notification.
import { workHourApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { WorkHourNotification } from '@/types/daily.types'
import { hhmm } from '@/utils/time'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitColumns, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const workHourCrud = dailyCrud(workHourApi)
const workHourConfig: CrudConfig<WorkHourNotification> = {
  title: 'WH & Notification',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'work-hour-notifications',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: workHourCrud.list,
  columns: [
    ...dateShiftUnitColumns<WorkHourNotification>(),
    col.text('category', 'Category'),
    col.text('sub_category', 'Sub Category'),
    col.clock('started_at', 'Start'),
    col.clock('ended_at', 'Stop'),
    col.number('total_hours', 'Total (jam)', 2),
    col.text('dedicated', 'Dedicated', { mobile: 'hide' }),
    col.text('remark', 'Remark', { mobile: 'hide' }),
  ],
  form: {
    width: 640,
    fields: [
      ...dateShiftUnitFields,
      { name: 'category', label: 'Category', required: true, placeholder: 'mis. Standby' },
      { name: 'sub_category', label: 'Sub Category', placeholder: 'mis. P5M' },
      { name: 'time_start', label: 'Jam Mulai', type: 'time', required: true },
      { name: 'time_end', label: 'Jam Selesai', type: 'time', required: true },
      { name: 'total_hours', label: 'Total', type: 'number', unit: 'jam', min: 0, max: 24, help: 'Kosong = dihitung dari jam mulai & selesai' },
      { name: 'dedicated', label: 'Dedicated', placeholder: 'mis. Waste Removal' },
      { name: 'remark', label: 'Remark', type: 'textarea', span: 'full' },
    ],
    toForm: (r) => ({ ...dailyToForm(r), time_start: hhmm(r.started_at), time_end: hhmm(r.ended_at) }),
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: workHourCrud.create,
    update: workHourCrud.update,
  },
  remove: { action: workHourCrud.remove, label: (r) => `${r.category} ${r.equipment_code}` },
  importType: 'work-hour-notifications',
  exportName: 'wh_notification',
}

export default function WhNotificationPage() {
  return <CrudPage config={workHourConfig} />
}
