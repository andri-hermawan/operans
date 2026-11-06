// System Data → Alert Rule.
import { Tag, Tooltip, Typography } from 'antd'

import { createAlertRule, deleteAlertRule, listAlertRules, updateAlertRule } from '@/services/api/alertRule.api'
import type { AlertRule } from '@/types/alertRule.types'
import type { CrudConfig } from '@/types/crud.types'
import { uses } from '@/utils/alertRules'
import { formatNumber } from '@/utils/format'
import { alertTypeOptions, recipientOptions, zoneNameOptions } from '@/utils/optionSources'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

const orDefault = (v: number | null, unit: string) => (v == null ? <Typography.Text type="secondary">Default</Typography.Text> : `${formatNumber(v, v % 1 ? 1 : 0)} ${unit}`)
const alertRuleConfig: CrudConfig<AlertRule> = {
  title: 'Alert Rule',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'alert-rules',
  subtitle: 'Aturan per project atau per zona. Isian kosong = memakai nilai bawaan Alert Category.',
  list: (_f, ctx) => listAlertRules(ctx.projectId),
  columns: [
    col.code('alert_type_name', 'Category', { mobile: 'title' }),
    { key: 'scope', title: 'Scope', render: (r) => <Tag color={r.zone_name ? 'purple' : 'blue'}>{r.zone_name ? 'Zona' : 'Project'}</Tag>, value: (r) => (r.zone_name ? 'Zona' : 'Project') },
    col.text('zone_name', 'Zona'),
    { key: 'duration_minutes', title: 'Duration', render: (r) => orDefault(r.duration_minutes, 'menit'), value: (r) => r.duration_minutes },
    { key: 'speed_limit_kph', title: 'Speed', render: (r) => orDefault(r.speed_limit_kph, 'km/h'), value: (r) => r.speed_limit_kph },
    { key: 'fuel_drop_liters', title: 'Fuel', render: (r) => orDefault(r.fuel_drop_liters, 'L'), value: (r) => r.fuel_drop_liters },
    {
      key: 'recipients',
      title: 'WhatsApp',
      render: (r) => <Tooltip title={r.recipients.map((x) => x.name).join(', ') || 'Tidak ada'}><Tag color={r.recipients.length ? 'green' : 'default'}>{r.recipients.length} penerima</Tag></Tooltip>,
      value: (r) => r.recipients.map((x) => x.name).join(', '),
    },
    col.text('description', 'Description', { mobile: 'hide' }),
    col.active(),
    col.dateTime('updated_at', 'Updated At', { mobile: 'hide' }),
  ],
  form: {
    width: 600,
    fields: [
      { name: 'alert_type', label: 'Category', type: 'select', options: alertTypeOptions, required: true, disabled: (_v, m) => m === 'edit' },
      { name: 'zone_name', label: 'Zona (kosong = seluruh project)', type: 'select', options: zoneNameOptions },
      { name: 'duration_minutes', label: 'Duration', type: 'number', unit: 'menit', min: 0, visible: (v) => uses(v.alert_type, 'duration') },
      { name: 'speed_limit_kph', label: 'Speed', type: 'number', unit: 'km/h', min: 0, visible: (v) => uses(v.alert_type, 'speed') },
      { name: 'fuel_drop_liters', label: 'Fuel', type: 'number', unit: 'L', min: 0, visible: (v) => uses(v.alert_type, 'fuel') },
      { name: 'recipient_user_ids', label: 'Penerima WhatsApp', type: 'multiSelect', options: recipientOptions, help: 'Hanya user aktif yang punya No. HP' },
      { name: 'description', label: 'Description', type: 'textarea', rules: [{ max: 255, message: 'Maks. 255 karakter' }] },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true },
    ],
    toForm: (r) => ({ ...r, recipient_user_ids: r.recipients.map((x) => x.user_id) }),
    create: (p, ctx) => createAlertRule(ruleBody(p, ctx.projectId)),
    update: (r, p) => updateAlertRule(r.id, ruleBody({ ...p, alert_type: r.alert_type })),
  },
  remove: { action: (r) => deleteAlertRule(r.id), label: (r) => `aturan ${r.alert_type_name}${r.zone_name ? ` (${r.zone_name})` : ''}` },
  exportName: 'alert_rule',
}
/** Ambang yang tidak dipakai jenis alert dikirim null. */
const ruleBody = (p: Record<string, unknown>, projectId?: string) => ({
  ...(projectId ? { project_id: projectId } : {}),
  alert_type: p.alert_type,
  zone_name: p.zone_name ?? null,
  duration_minutes: uses(p.alert_type, 'duration') ? (p.duration_minutes ?? null) : null,
  speed_limit_kph: uses(p.alert_type, 'speed') ? (p.speed_limit_kph ?? null) : null,
  fuel_drop_liters: uses(p.alert_type, 'fuel') ? (p.fuel_drop_liters ?? null) : null,
  description: p.description ?? null,
  is_active: p.is_active ?? true,
  recipient_user_ids: p.recipient_user_ids ?? [],
})

export default function AlertRulePage() {
  return <CrudPage config={alertRuleConfig} />
}
