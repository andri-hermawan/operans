// Record / Upload → Transaction → Time Sheet.
import { timeSheetApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { TimeSheet } from '@/types/daily.types'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitColumns, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const timeSheetCrud = dailyCrud(timeSheetApi)
const timeSheetConfig: CrudConfig<TimeSheet> = {
  title: 'Time Sheet',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'time-sheets',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: timeSheetCrud.list,
  columns: [
    ...dateShiftUnitColumns<TimeSheet>(),
    col.text('operator_nik', 'NIK Operator', { mobile: 'hide' }),
    col.text('operator_name', 'Operator'),
    col.number('hm_start', 'HM Start', 2),
    col.number('hm_end', 'HM End', 2),
    col.number('hm_total', 'Total (jam)', 2),
    col.text('remark', 'Remark', { mobile: 'hide' }),
  ],
  form: {
    width: 640,
    fields: [
      ...dateShiftUnitFields,
      { name: 'operator_nik', label: 'NIK Operator' },
      { name: 'operator_name', label: 'Operator' },
      { name: 'hm_start', label: 'HM Start', type: 'number', unit: 'jam', min: 0 },
      { name: 'hm_end', label: 'HM End', type: 'number', unit: 'jam', min: 0 },
      { name: 'hm_total', label: 'Total', type: 'number', unit: 'jam', min: 0, help: 'Kosong = HM End − HM Start (dihitung otomatis)' },
      { name: 'remark', label: 'Remark', type: 'textarea', span: 'full' },
    ],
    toForm: dailyToForm,
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: timeSheetCrud.create,
    update: timeSheetCrud.update,
  },
  remove: { action: timeSheetCrud.remove, label: (r) => `time sheet ${r.equipment_code}` },
  importType: 'time-sheets',
  exportName: 'time_sheet',
}

export default function TimeSheetPage() {
  return <CrudPage config={timeSheetConfig} />
}
