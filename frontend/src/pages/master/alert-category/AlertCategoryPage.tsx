// System Data → Alert Category (jenis alert, tetap 5 jenis).
import { listAlertTypes, updateAlertType } from '@/services/api/alertType.api'
import type { AlertType } from '@/types/alertRule.types'
import type { CrudConfig } from '@/types/crud.types'
import { uses } from '@/utils/alertRules'
import { formatNumber } from '@/utils/format'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const thresholdText = (v: number | null, unit: string) => (v == null ? '-' : `${formatNumber(v, v % 1 ? 1 : 0)} ${unit}`)
const alertTypeConfig: CrudConfig<AlertType> = {
  title: 'Alert Category',
  resource: 'master',
  rowKey: 'code',
  queryKey: 'alert-types',
  subtitle: 'Jenis alert & ambang bawaan. Ambang khusus project/zona diatur di Alert Rule. Hanya Super Admin yang dapat mengubah.',
  list: listAlertTypes,
  columns: [
    col.code('legacy_code', 'Code', { mobile: 'title' }),
    col.text('name', 'Name'),
    col.text('description', 'Description'),
    { key: 'default_duration_minutes', title: 'Durasi bawaan', render: (r) => (r.uses_duration ? thresholdText(r.default_duration_minutes, 'menit') : '-') },
    { key: 'default_speed_limit_kph', title: 'Speed bawaan', render: (r) => (r.uses_speed_limit ? thresholdText(r.default_speed_limit_kph, 'km/h') : '-') },
    { key: 'default_fuel_drop_liters', title: 'Fuel bawaan', render: (r) => (r.uses_fuel_drop ? thresholdText(r.default_fuel_drop_liters, 'L') : '-') },
  ],
  form: {
    fields: [
      { name: 'code', label: 'Kode', disabled: () => true },
      { name: 'name', label: 'Name', required: true },
      { name: 'description', label: 'Description', type: 'textarea' },
      { name: 'default_duration_minutes', label: 'Durasi bawaan', type: 'number', unit: 'menit', min: 0, visible: (v) => uses(v.code, 'duration') },
      { name: 'default_speed_limit_kph', label: 'Speed bawaan', type: 'number', unit: 'km/h', min: 0, visible: (v) => uses(v.code, 'speed') },
      { name: 'default_fuel_drop_liters', label: 'Fuel bawaan', type: 'number', unit: 'L', min: 0, visible: (v) => uses(v.code, 'fuel') },
    ],
    canEdit: (_r, ctx) => ctx.role === 'superadmin',
    update: (r, p) => {
      const body: Record<string, unknown> = { name: p.name, description: p.description ?? '' }
      for (const k of ['default_duration_minutes', 'default_speed_limit_kph', 'default_fuel_drop_liters']) if (k in p) body[k] = p[k]
      return updateAlertType(r.code, body)
    },
  },
}

export default function AlertCategoryPage() {
  return <CrudPage config={alertTypeConfig} />
}
