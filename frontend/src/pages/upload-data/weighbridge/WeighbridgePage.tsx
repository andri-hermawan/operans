// Record / Upload → Weighbridge.
import { Typography } from 'antd'

import { ticketApi } from '@/services/api/daily.api'
import type { CrudConfig } from '@/types/crud.types'
import type { WeighbridgeTicket } from '@/types/daily.types'
import { formatNumber } from '@/utils/format'
import { hhmm } from '@/utils/time'
import CrudPage from '@/components/crud/CrudPage'
import { dailyCrud, dailyDefaults, dailyFilters, dailyToForm, dateShiftUnitFields } from '@/components/crud/dailyConfig'
import { col } from '@/components/table/columns'

const ticketCrud = dailyCrud(ticketApi)
const weighbridgeConfig: CrudConfig<WeighbridgeTicket> = {
  title: 'Weighbridge',
  resource: 'daily',
  rowKey: 'id',
  queryKey: 'weighbridge-tickets',
  filters: dailyFilters,
  defaultFilters: dailyDefaults,
  list: ticketCrud.list,
  columns: [
    col.rowNo(),
    col.plainDate('operational_date', 'Date'),
    col.text('shift_name', 'Shift'),
    col.code('ticket_no', 'Ticket No', { mobile: 'title' }),
    col.code('equipment_code', 'Asset ID'),
    col.text('product', 'Product'),
    col.number('gross_kg', 'Gross (kg)'),
    col.number('tare_kg', 'Tare (kg)'),
    col.number('net_kg', 'Net (kg)'),
    col.text('recipient', 'Recipient', { mobile: 'hide' }),
    col.text('customer', 'Customer', { mobile: 'hide' }),
    col.text('transporter', 'Transporter', { mobile: 'hide' }),
    col.clock('gross_at', 'Gross Time'),
    col.clock('tare_at', 'Tare Time'),
    col.text('gross_operator', 'Gross Operator', { mobile: 'hide' }),
    col.text('tare_operator', 'Tare Operator', { mobile: 'hide' }),
    col.text('description', 'Description', { mobile: 'hide' }),
    col.text('location', 'Location', { mobile: 'hide' }),
  ],
  summary: (rows) => (
    <Typography.Text>
      Total Net: <b>{formatNumber(rows.reduce((s, r) => s + r.net_kg, 0))} kg</b> · {rows.length} tiket
    </Typography.Text>
  ),
  form: {
    width: 720,
    fields: [
      ...dateShiftUnitFields,
      { name: 'ticket_no', label: 'Ticket No', required: true },
      { name: 'product', label: 'Product' },
      { name: 'gross_kg', label: 'Gross', type: 'number', unit: 'kg', min: 0, required: true },
      { name: 'tare_kg', label: 'Tare', type: 'number', unit: 'kg', min: 0, required: true, help: 'Net = Gross − Tare (dihitung otomatis)' },
      { name: 'gross_time', label: 'Gross Time', type: 'time' },
      { name: 'tare_time', label: 'Tare Time', type: 'time' },
      { name: 'gross_operator', label: 'Gross Operator' },
      { name: 'tare_operator', label: 'Tare Operator' },
      { name: 'recipient', label: 'Recipient' },
      { name: 'customer', label: 'Customer' },
      { name: 'transporter', label: 'Transporter' },
      { name: 'location', label: 'Location' },
      { name: 'description', label: 'Description', type: 'textarea' },
    ],
    toForm: (r) => ({ ...dailyToForm(r), gross_time: hhmm(r.gross_at), tare_time: hhmm(r.tare_at) }),
    initial: (_c, f) => ({ operational_date: f.date, shift: f.shift }),
    create: ticketCrud.create,
    update: ticketCrud.update,
  },
  remove: { action: ticketCrud.remove, label: (r) => `tiket ${r.ticket_no}` },
  importType: 'weighbridge-tickets',
  exportName: 'weighbridge',
}

export default function WeighbridgePage() {
  return <CrudPage config={weighbridgeConfig} />
}
