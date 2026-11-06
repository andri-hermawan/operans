// System Data → Report Schedule: kirim laporan PDF otomatis lewat email (core-api membuat PDF dari halaman cetak).
import { Tag } from 'antd'

import { createReportSchedule, deleteReportSchedule, listReportSchedules, updateReportSchedule } from '@/services/api/reportSchedule.api'
import { listShifts } from '@/services/api/shift.api'
import { listUsers } from '@/services/api/user.api'
import type { CrudConfig } from '@/types/crud.types'
import type { Schedule } from '@/types/reportSchedule.types'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

import ScheduleActions from './components/ScheduleActions'

const REPORT_LABEL: Record<string, string> = {
  alert_summary: 'Alert Summary',
  executive_hauling: 'Executive Summary Report Hauling',
  executive_emplacement: 'Executive Summary Report Emplacement',
}
const ALL_DAY = '__day'

const config: CrudConfig<Schedule> = {
  title: 'Report Schedule',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'report-schedules',
  subtitle: 'Laporan dibuat PDF lalu dikirim email setiap hari pada jam kirim (jam project). Email server diatur di .env core-api (SMTP_*).',
  list: (_f, ctx) => listReportSchedules(ctx.projectId),
  columns: [
    col.code('name', 'Name', { mobile: 'title' }),
    { key: 'report_type', title: 'Report', render: (r) => REPORT_LABEL[r.report_type] ?? r.report_type, value: (r) => REPORT_LABEL[r.report_type] ?? r.report_type },
    { key: 'shift_code', title: 'Shift', render: (r) => r.shift_code ?? 'Semua shift', value: (r) => r.shift_code ?? 'Semua shift' },
    { key: 'date_offset', title: 'Data', render: (r) => (r.date_offset === -1 ? 'H-1' : 'Hari ini'), value: (r) => (r.date_offset === -1 ? 'H-1' : 'Hari ini') },
    col.text('send_time', 'Jam Kirim'),
    { key: 'recipients', title: 'Recipient', render: (r) => r.recipients.map((x) => <Tag key={x.user_id}>{x.name}</Tag>), value: (r) => r.recipients.map((x) => x.email).join(', ') },
    col.active(),
    col.dateTime('last_run_at', 'Terakhir Kirim'),
  ],
  form: {
    fields: [
      { name: 'name', label: 'Nama jadwal', required: true, span: 'full' },
      { name: 'report_type', label: 'Report', type: 'select', required: true, span: 'full', initial: 'alert_summary', options: Object.entries(REPORT_LABEL).map(([value, label]) => ({ value, label })) },
      { name: 'shift_code', label: 'Shift', type: 'select', initial: ALL_DAY, options: { key: 'schedule-shifts', load: async (pid) => [{ value: ALL_DAY, label: 'Semua shift (satu hari)' }, ...(await listShifts(pid)).map((s) => ({ value: s.shift_code, label: s.shift_name }))] } },
      { name: 'date_offset', label: 'Data', type: 'radio', initial: -1, options: [{ value: '-1', label: 'H-1 (kemarin)' }, { value: '0', label: 'Hari ini' }] },
      { name: 'send_time', label: 'Jam Kirim', type: 'time', required: true, initial: '09:00' },
      { name: 'recipient_user_ids', label: 'Penerima (email)', type: 'multiSelect', required: true, options: { key: 'schedule-users', load: async (pid) => (await listUsers(pid)).filter((u) => u.is_active).map((u) => ({ value: u.id, label: `${u.name} <${u.email}>` })) } },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    toForm: (r) => ({ ...r, shift_code: r.shift_code ?? ALL_DAY, date_offset: String(r.date_offset) }),
    initial: () => ({ date_offset: '-1' }),
    create: (p, ctx) => createReportSchedule(body(p, ctx.projectId)),
    update: (r, p) => updateReportSchedule(r.id, body(p)),
  },
  remove: { action: (r) => deleteReportSchedule(r.id), label: (r) => `jadwal ${r.name}` },
  rowActions: (r) => <ScheduleActions s={r} />,
}
const body = (p: Record<string, unknown>, projectId?: string) => ({
  ...(projectId ? { project_id: projectId } : {}),
  name: p.name,
  report_type: p.report_type,
  shift_code: p.shift_code === ALL_DAY || !p.shift_code ? '' : p.shift_code,
  date_offset: Number(p.date_offset ?? -1),
  send_time: p.send_time,
  recipient_user_ids: p.recipient_user_ids ?? [],
  is_active: p.is_active ?? true,
})

export default function ReportSchedulePage() {
  return <CrudPage config={config} />
}
