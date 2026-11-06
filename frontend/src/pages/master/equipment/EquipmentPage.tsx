// System Data → Equipment.
import { createEquipment, listEquipment, updateEquipment } from '@/services/api/equipment.api'
import type { CrudConfig } from '@/types/crud.types'
import type { Equipment } from '@/types/equipment.types'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const EQUIPMENT_TYPES = ['Dump Truck', 'Light Vehicle', 'Excavator', 'Dozer', 'Grader', 'Water Truck', 'Fuel Truck'].map((v) => ({ value: v, label: v }))
const BRANDS = ['Hino', 'Renault', 'Mitsubishi', 'Isuzu', 'Fuso', 'Volvo', 'Scania', 'Mercedes-Benz', 'Toyota'].map((v) => ({ value: v, label: v }))
const equipmentConfig: CrudConfig<Equipment> = {
  title: 'Equipment',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'equipment-master',
  list: (_f, ctx) => listEquipment(ctx.projectId, true),
  columns: [
    col.code('equipment_code', 'Unit Code', { mobile: 'title' }),
    col.text('equipment_alias', 'Alias'),
    col.text('equipment_type', 'Type'),
    col.text('brand', 'Brand'),
    col.text('model', 'Model'),
    col.text('class', 'Class'),
    col.active(),
    col.dateTime('created_at', 'Created At', { mobile: 'hide' }),
  ],
  form: {
    width: 560,
    fields: [
      { name: 'equipment_code', label: 'Unit Code', type: 'upper', required: true },
      { name: 'equipment_alias', label: 'Alias' },
      { name: 'equipment_type', label: 'Type', type: 'select', options: EQUIPMENT_TYPES },
      { name: 'brand', label: 'Brand', type: 'select', options: BRANDS },
      { name: 'model', label: 'Model' },
      { name: 'class', label: 'Class' },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    create: (p, ctx) => createEquipment({ ...p, project_id: ctx.projectId }),
    update: (r, p) => updateEquipment(r.id, p),
  },
  exportName: 'equipment',
}

export default function EquipmentPage() {
  return <CrudPage config={equipmentConfig} />
}
