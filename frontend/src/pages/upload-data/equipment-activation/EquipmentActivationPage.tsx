// Record / Upload → Equipment Activation.
import { Tag } from 'antd'

import { activationApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { EquipmentActivation } from '@/types/daily.types'
import CrudPage from '@/components/crud/CrudPage'
import { dailyDefaults } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const activationConfig: CrudConfig<EquipmentActivation> = {
  title: 'Equipment Activation',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'equipment-activations',
  filters: [{ type: 'date' }],
  defaultFilters: dailyDefaults,
  list: (f, ctx) => activationApi.list(ctx.projectId, f.date),
  columns: [
    col.rowNo(),
    col.plainDate('activation_date', 'Date'),
    col.code('equipment_code', 'Asset ID', { mobile: 'title' }),
    { key: 'status', title: 'Status', render: (r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? 'Active' : 'Non Active'}</Tag>, value: (r) => (r.status === 'active' ? 'Active' : 'Non Active') },
    col.text('description', 'Description'),
  ],
  form: {
    fields: [
      { name: 'activation_date', label: 'Tanggal', type: 'date', required: true },
      { name: 'equipment_id', label: 'Equipment', type: 'select', options: 'equipmentIds', required: true, disabled: (_v, mode) => mode === 'edit' },
      { name: 'status', label: 'Status', type: 'radio', required: true, initial: 'active', options: [{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Non Active' }] },
      { name: 'description', label: 'Description', type: 'textarea' },
    ],
    initial: (_c, f) => ({ activation_date: f.date }),
    create: (p) => activationApi.create(p),
    update: (r, p) => activationApi.update(r.id, { activation_date: p.activation_date, status: p.status, description: p.description }),
  },
  remove: { action: (r) => activationApi.remove(r.id), label: (r) => `aktivasi ${r.equipment_code}` },
  importType: 'equipment-activations',
  exportName: 'equipment_activation',
}

export default function EquipmentActivationPage() {
  return <CrudPage config={activationConfig} />
}
