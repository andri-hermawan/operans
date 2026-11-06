// Record / Upload → Transaction → Rain & Slippery.
import { rainSlipperyApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { RainSlipperyLog } from '@/types/daily.types'
import { hhmm } from '@/utils/time'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const rainSlipperyCrud = dailyCrud(rainSlipperyApi)
const rainSlipperyConfig: CrudConfig<RainSlipperyLog> = {
  title: 'Rain & Slippery',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'rain-slippery-logs',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: rainSlipperyCrud.list,
  columns: [
    col.rowNo(),
    col.plainDate('operational_date', 'Date'),
    col.text('shift_name', 'Shift'),
    col.text('location', 'Location', { mobile: 'title' }),
    col.clock('rain_start_at', 'Start Rain'),
    col.clock('rain_stop_at', 'Stop Rain'),
    col.number('rain_hours', 'Total Rain', 2),
    col.clock('wait_slippery_start_at', 'Start Wait Slippery', { mobile: 'hide' }),
    col.clock('wait_slippery_stop_at', 'Stop Wait Slippery', { mobile: 'hide' }),
    col.number('wait_slippery_hours', 'Total Wait Slippery', 2, { mobile: 'hide' }),
    col.clock('slippery_start_at', 'Start Slippery', { mobile: 'hide' }),
    col.clock('slippery_stop_at', 'Stop Slippery', { mobile: 'hide' }),
    col.number('slippery_hours', 'Total Slippery', 2),
    col.number('rainfall_mm', 'Rain Fall (mm)', 2),
  ],
  form: {
    width: 720,
    fields: [
      ...dateShiftUnitFields.slice(0, 2),
      { name: 'location', label: 'Location', span: 'full' },
      { name: 'rain_start', label: 'Start Rain', type: 'time' },
      { name: 'rain_stop', label: 'Stop Rain', type: 'time' },
      { name: 'rain_hours', label: 'Total Rain', type: 'number', unit: 'jam', min: 0, max: 24, help: 'Kosong = dihitung dari jam' },
      { name: 'wait_slippery_start', label: 'Start Wait Slippery', type: 'time' },
      { name: 'wait_slippery_stop', label: 'Stop Wait Slippery', type: 'time' },
      { name: 'wait_slippery_hours', label: 'Total Wait Slippery', type: 'number', unit: 'jam', min: 0, max: 24 },
      { name: 'slippery_start', label: 'Start Slippery', type: 'time' },
      { name: 'slippery_stop', label: 'Stop Slippery', type: 'time' },
      { name: 'slippery_hours', label: 'Total Slippery', type: 'number', unit: 'jam', min: 0, max: 24 },
      { name: 'rainfall_mm', label: 'Rain Fall', type: 'number', unit: 'mm', min: 0 },
    ],
    toForm: (r) => ({
      ...dailyToForm(r),
      rain_start: hhmm(r.rain_start_at),
      rain_stop: hhmm(r.rain_stop_at),
      wait_slippery_start: hhmm(r.wait_slippery_start_at),
      wait_slippery_stop: hhmm(r.wait_slippery_stop_at),
      slippery_start: hhmm(r.slippery_start_at),
      slippery_stop: hhmm(r.slippery_stop_at),
    }),
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: rainSlipperyCrud.create,
    update: rainSlipperyCrud.update,
  },
  remove: { action: rainSlipperyCrud.remove, label: (r) => `hujan & licin ${r.location ?? r.operational_date}` },
  importType: 'rain-slippery-logs',
  exportName: 'rain_slippery',
}

export default function RainSlipperyPage() {
  return <CrudPage config={rainSlipperyConfig} />
}
