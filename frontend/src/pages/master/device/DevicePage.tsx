// System Data → Device (alat GPS, layanan IoT).
import { Typography } from 'antd'

import { createDevice, listDevices, updateDevice } from '@/services/api/device.api'
import type { CrudConfig } from '@/types/crud.types'
import type { Device } from '@/types/device.types'
import { formatDateTime } from '@/utils/format'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

import ApiKeyButton from './components/ApiKeyButton'

const MODELS = ['FMC125', 'FMC150', 'simulator'].map((v) => ({ value: v, label: v }))
const FUEL_SOURCES = [
  { value: 'escort_ble', label: 'Escort BLE (LLS)' },
  { value: 'ecu', label: 'ECU / CAN' },
]
const deviceConfig: CrudConfig<Device> = {
  title: 'Device',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'devices',
  subtitle: 'Alat GPS Teltonika. IMEI tidak bisa diubah; satu unit hanya boleh memakai satu alat aktif.',
  list: listDevices,
  columns: [
    { key: 'imei', title: 'IMEI', fixed: 'left', mobile: 'title', render: (r) => <Typography.Text code>{r.imei}</Typography.Text>, value: (r) => r.imei },
    col.text('name', 'Device Name'),
    col.text('provider_name', 'Provider'),
    col.text('sim_number', 'SIM'),
    col.text('model', 'Model'),
    { key: 'equipment_code', title: 'Equipment', render: (r) => r.equipment_code ?? <Typography.Text type="secondary">Belum dipasang</Typography.Text>, value: (r) => r.equipment_code },
    { key: 'last_event_at', title: 'Data Terakhir', render: (r) => formatDateTime(r.last_event_at), value: (r) => formatDateTime(r.last_event_at) },
    col.active(),
  ],
  form: {
    fields: [
      { name: 'imei', label: 'IMEI', required: true, rules: [{ pattern: /^\d{15}$/, message: 'IMEI 15 digit angka' }], disabled: (_v, m) => m === 'edit' },
      { name: 'name', label: 'Device Name' },
      { name: 'equipment_id', label: 'Equipment', type: 'select', options: 'equipmentIds', help: 'Kosongkan untuk melepas alat dari unit' },
      { name: 'model', label: 'Model', type: 'select', options: MODELS },
      { name: 'fuel_source', label: 'Sumber fuel', type: 'select', options: FUEL_SOURCES },
      { name: 'provider_name', label: 'Provider SIM' },
      { name: 'sim_number', label: 'No. SIM' },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    create: (p) => createDevice(p),
    update: (r, p) =>
      updateDevice(r.id, {
        ...p,
        imei: undefined,
        equipment_id: p.equipment_id ?? undefined,
        clear_equipment: !p.equipment_id && !!r.equipment_id,
        fuel_source: p.fuel_source ?? '',
      }),
  },
  rowActions: (r) => <ApiKeyButton device={r} />,
  exportName: 'device',
}

export default function DevicePage() {
  return <CrudPage config={deviceConfig} />
}
