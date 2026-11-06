// Record / Upload → Setting Operator.
import { operatorApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { OperatorAssignment } from '@/types/daily.types'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitColumns, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const operatorCrud = dailyCrud(operatorApi)
const operatorConfig: CrudConfig<OperatorAssignment> = {
  title: 'Setting Operator',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'operator-assignments',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: operatorCrud.list,
  columns: [...dateShiftUnitColumns<OperatorAssignment>(), col.text('operator_name', 'Operator Name'), col.text('description', 'Description')],
  form: {
    fields: [...dateShiftUnitFields, { name: 'operator_name', label: 'Operator Name', required: true }, { name: 'description', label: 'Description', type: 'textarea' }],
    toForm: dailyToForm,
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: operatorCrud.create,
    update: operatorCrud.update,
  },
  remove: { action: operatorCrud.remove, label: (r) => `operator ${r.equipment_code} (${r.operator_name})` },
  importType: 'operator-assignments',
  exportName: 'setting_operator',
}

export default function DailySettingOperatorPage() {
  return <CrudPage config={operatorConfig} />
}
