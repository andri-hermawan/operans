// Record / Upload → Transaction → Ritase.
import { Typography } from 'antd'

import { ritaseApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { Ritase } from '@/types/daily.types'
import { formatNumber } from '@/utils/format'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const ritaseCrud = dailyCrud(ritaseApi)
const ritaseConfig: CrudConfig<Ritase> = {
  title: 'Ritase',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'ritases',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: ritaseCrud.list,
  columns: [
    col.rowNo(),
    col.plainDate('operational_date', 'Date'),
    col.text('shift_name', 'Shift'),
    col.text('time_slot', 'Time'),
    col.code('loader_code', 'Loader', { mobile: 'title' }),
    col.text('loader_operator_name', 'Operator Loader', { mobile: 'hide' }),
    col.text('loader_partner', 'User Loader', { mobile: 'hide' }),
    col.code('hauler_code', 'Hauler'),
    col.text('hauler_operator_name', 'Operator Hauler', { mobile: 'hide' }),
    col.text('hauler_partner', 'User Hauler', { mobile: 'hide' }),
    col.text('block', 'Blok', { mobile: 'hide' }),
    col.text('location', 'Location', { mobile: 'hide' }),
    col.text('dumping', 'Dumping', { mobile: 'hide' }),
    col.text('material', 'Material'),
    col.text('sub_material', 'Sub Material', { mobile: 'hide' }),
    col.text('seam', 'Seam', { mobile: 'hide' }),
    col.text('phase', 'Phase', { mobile: 'hide' }),
    col.number('distance', 'Distance', 2),
    col.number('ritase', 'Ritase', 2),
    col.number('production', 'Prod', 2),
    col.number('production_adjusted', 'Prod Adj', 2, { mobile: 'hide' }),
    col.number('distance_volume', 'Dist x Vol', 2, { mobile: 'hide' }),
    col.number('productivity_factor', 'Productivity Factor', 4, { mobile: 'hide' }),
  ],
  summary: (rows) => (
    <Typography.Text>
      Total Ritase: <b>{formatNumber(rows.reduce((s, r) => s + (r.ritase ?? 0), 0))}</b> · Total Prod:{' '}
      <b>{formatNumber(rows.reduce((s, r) => s + (r.production ?? 0), 0))}</b> · {rows.length} baris
    </Typography.Text>
  ),
  form: {
    width: 880,
    fields: [
      ...dateShiftUnitFields.slice(0, 2),
      { name: 'time_slot', label: 'Time', required: true, placeholder: '07.00-08.00' },
      { name: 'loader_code', label: 'Loader', type: 'combo', options: 'equipmentCodes', required: true },
      { name: 'loader_model', label: 'Model Loader' },
      { name: 'loader_operator_nik', label: 'NIK Operator Loader' },
      { name: 'loader_operator_name', label: 'Operator Loader' },
      { name: 'loader_partner', label: 'User Loader', help: 'Kode mitra, mis. TBBE' },
      { name: 'hauler_code', label: 'Hauler', type: 'combo', options: 'equipmentCodes', required: true },
      { name: 'hauler_model', label: 'Model Hauler' },
      { name: 'hauler_operator_nik', label: 'NIK Operator Hauler' },
      { name: 'hauler_operator_name', label: 'Operator Hauler' },
      { name: 'hauler_partner', label: 'User Hauler' },
      { name: 'block', label: 'Blok' },
      { name: 'location', label: 'Location' },
      { name: 'dumping', label: 'Dumping' },
      { name: 'material', label: 'Material' },
      { name: 'sub_material', label: 'Sub Material' },
      { name: 'seam', label: 'Seam' },
      { name: 'phase', label: 'Phase' },
      { name: 'distance', label: 'Distance', type: 'number', min: 0 },
      { name: 'ritase', label: 'Ritase', type: 'number', min: 0 },
      { name: 'production', label: 'Prod', type: 'number', min: 0 },
      { name: 'production_adjusted', label: 'Prod Adj', type: 'number', min: 0 },
      { name: 'distance_volume', label: 'Dist x Vol', type: 'number', min: 0 },
      { name: 'productivity_factor', label: 'Productivity Factor', type: 'number', min: 0 },
    ],
    toForm: dailyToForm,
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: ritaseCrud.create,
    update: ritaseCrud.update,
  },
  remove: { action: ritaseCrud.remove, label: (r) => `ritase ${r.loader_code} → ${r.hauler_code}` },
  importType: 'ritases',
  exportName: 'ritase',
}

export default function RitasePage() {
  return <CrudPage config={ritaseConfig} />
}
