// System Data → Shift.
import { Tag } from 'antd'

import { createShift, listShiftSettings, updateShift } from '@/services/api/shift.api'
import type { CrudConfig } from '@/types/crud.types'
import type { ShiftRow } from '@/types/shift.types'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const crosses = (r: ShiftRow) => r.end_time <= r.start_time
const shiftConfig: CrudConfig<ShiftRow> = {
  title: 'Shift',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'shifts-manage',
  subtitle: 'Jam selesai lebih kecil dari jam mulai = shift melewati tengah malam (data sebelum jam selesai masuk tanggal operasional kemarin).',
  list: (_f, ctx) => listShiftSettings(ctx.projectId),
  columns: [
    col.number('sequence', 'Urutan'),
    col.code('shift_code', 'Code', { mobile: 'title' }),
    col.text('shift_name', 'Name'),
    { key: 'hours', title: 'Jam', render: (r) => <>{r.start_time} – {r.end_time} {crosses(r) && <Tag color="purple">lewat tengah malam</Tag>}</>, value: (r) => `${r.start_time}-${r.end_time}` },
    col.active(),
    col.dateTime('updated_at', 'Updated At', { mobile: 'hide' }),
  ],
  form: {
    fields: [
      { name: 'shift_code', label: 'Code', type: 'upper', required: true, placeholder: 'SHIFT_1' },
      { name: 'shift_name', label: 'Name', required: true, placeholder: 'Shift 1' },
      { name: 'start_time', label: 'Jam Mulai', type: 'time', required: true },
      { name: 'end_time', label: 'Jam Selesai', type: 'time', required: true },
      { name: 'sequence', label: 'Urutan', type: 'number', min: 1, max: 9, required: true },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    create: (p, ctx) => createShift({ ...p, project_id: ctx.projectId }),
    update: (r, p) => updateShift(r.id, p),
  },
  exportName: 'shift',
}

export default function ShiftPage() {
  return <CrudPage config={shiftConfig} />
}
