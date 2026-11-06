// System Data → Fuel Calibration.
import { Tag } from 'antd'

import { deleteFuelCalibration, listFuelCalibrations, saveFuelCalibration } from '@/services/api/fuelCalibration.api'
import type { CrudConfig } from '@/types/crud.types'
import type { FuelCalibration } from '@/types/fuelCalibration.types'
import { formatNumber } from '@/utils/format'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const fuelCalibrationConfig: CrudConfig<FuelCalibration> = {
  title: 'Fuel Calibration',
  resource: 'master',
  rowKey: 'equipment_id',
  queryKey: 'fuel-calibrations',
  subtitle: 'Sensor LLS: liter dihitung dari nilai sensor (0 … nilai maks) sampai kapasitas tangki. Percentage: alat mengirim persen.',
  list: (_f, ctx) => listFuelCalibrations(ctx.projectId),
  columns: [
    col.code('equipment_code', 'Equipment Code', { mobile: 'title' }),
    col.number('tank_capacity_liters', 'Max Fuel Volume (L)', 0),
    { key: 'calibration_type', title: 'Type', render: (r) => <Tag color={r.calibration_type === 'lls' ? 'blue' : 'purple'}>{r.calibration_type === 'lls' ? 'LLS' : 'Percentage'}</Tag>, value: (r) => r.calibration_type.toUpperCase() },
    { key: 'max_sensor_value', title: 'Max Fuel Level', align: 'right', render: (r) => (r.calibration_type === 'percentage' ? `${formatNumber(r.max_sensor_value)}%` : formatNumber(r.max_sensor_value)) },
    { key: 'point_count', title: 'Titik', align: 'right', render: (r) => `${r.point_count}${r.is_linear ? '' : ' (tabel uji)'}` },
    col.dateTime('updated_at', 'Updated At', { mobile: 'hide' }),
  ],
  form: {
    fields: [
      { name: 'equipment_id', label: 'Equipment', type: 'select', options: 'equipmentIds', required: true, disabled: (_v, m) => m === 'edit' },
      { name: 'calibration_type', label: 'Type', type: 'radio', required: true, initial: 'lls', options: [{ value: 'lls', label: 'LLS' }, { value: 'percentage', label: 'Percentage' }] },
      { name: 'tank_capacity_liters', label: 'Max Fuel Volume', type: 'number', unit: 'L', min: 1, required: true },
      { name: 'max_sensor_value', label: 'Max Fuel Level (nilai sensor)', type: 'number', min: 1, required: true, visible: (v) => v.calibration_type !== 'percentage' },
    ],
    create: (p) => saveCalibration(p),
    update: (_r, p) => saveCalibration(p),
  },
  remove: { action: (r) => deleteFuelCalibration(r.equipment_id), label: (r) => `kalibrasi ${r.equipment_code}` },
  exportName: 'fuel_calibration',
}
const saveCalibration = (p: Record<string, unknown>) =>
  saveFuelCalibration(String(p.equipment_id), {
    calibration_type: p.calibration_type,
    tank_capacity_liters: p.tank_capacity_liters,
    max_sensor_value: p.calibration_type === 'percentage' ? 100 : p.max_sensor_value,
  })

export default function FuelCalibrationPage() {
  return <CrudPage config={fuelCalibrationConfig} />
}
