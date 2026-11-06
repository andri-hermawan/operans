import { useState } from 'react'
import { App, Button, Modal, Table, Tag, Tooltip } from 'antd'
import { Eye, History, Send } from 'lucide-react'

import { listScheduleLogs, previewSchedulePdf, sendScheduleNow } from '@/services/api/reportSchedule.api'
import { errorMessage } from '@/services/http/ApiError'
import type { Schedule, ScheduleLog } from '@/types/reportSchedule.types'
import { formatDateTime, formatPlainDate } from '@/utils/format'

export default function ScheduleActions({ s }: { s: Schedule }) {
  const { message, modal } = App.useApp()
  const [logs, setLogs] = useState<ScheduleLog[]>()
  const [busy, setBusy] = useState<'send' | 'preview'>()

  const openLogs = async () => {
    try {
      setLogs(await listScheduleLogs(s.id))
    } catch (e) {
      message.error(errorMessage(e))
    }
  }
  const sendNow = () =>
    modal.confirm({
      title: `Kirim "${s.name}" sekarang?`,
      content: `Ke ${s.recipients.length} penerima. Membuat PDF butuh ± 10–30 detik.`,
      okText: 'Kirim',
      onOk: async () => {
        setBusy('send')
        try {
          await sendScheduleNow(s.id)
          message.success('Laporan terkirim')
        } catch (e) {
          message.error(errorMessage(e))
        } finally {
          setBusy(undefined)
        }
      },
    })
  const preview = async () => {
    setBusy('preview')
    try {
      const pdf = await previewSchedulePdf(s.id)
      window.open(URL.createObjectURL(pdf), '_blank')
    } catch (e) {
      message.error(errorMessage(e))
    } finally {
      setBusy(undefined)
    }
  }

  return (
    <>
      <Tooltip title="Riwayat kirim"><Button size="small" type="text" icon={<History size={15} />} onClick={openLogs} /></Tooltip>
      <Tooltip title="Lihat PDF (tanpa kirim)"><Button size="small" type="text" icon={<Eye size={15} />} loading={busy === 'preview'} onClick={preview} /></Tooltip>
      <Tooltip title="Kirim sekarang"><Button size="small" type="text" icon={<Send size={15} />} loading={busy === 'send'} onClick={sendNow} /></Tooltip>
      <Modal open={!!logs} title={`Riwayat — ${s.name}`} footer={null} width={780} onCancel={() => setLogs(undefined)}>
        <Table<ScheduleLog>
          size="small"
          rowKey="id"
          dataSource={logs}
          pagination={{ pageSize: 10, size: 'small' }}
          scroll={{ x: 'max-content' }}
          columns={[
            { title: 'Waktu', dataIndex: 'created_at', render: (v: string) => formatDateTime(v) },
            { title: 'Data', dataIndex: 'report_date', render: (v: string) => formatPlainDate(v) },
            { title: 'Trigger', dataIndex: 'trigger', render: (v: string) => (v === 'manual' ? 'Manual' : 'Jadwal') },
            { title: 'Status', dataIndex: 'status', render: (v: string) => <Tag color={v === 'sent' ? 'success' : 'error'}>{v === 'sent' ? 'Terkirim' : 'Gagal'}</Tag> },
            { title: 'Penerima', dataIndex: 'recipients', align: 'right' },
            { title: 'Error', dataIndex: 'error_message' },
          ]}
        />
      </Modal>
    </>
  )
}
