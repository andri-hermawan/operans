// Jadwal kirim laporan PDF lewat email (core-api) dan riwayat kirimnya.

export interface Schedule {
  id: string
  name: string
  report_type: string
  shift_code: string | null
  date_offset: number
  send_time: string
  recipient_user_ids: string[]
  recipients: { user_id: string; name: string; email: string }[]
  is_active: boolean
  last_run_at: string | null
}
export interface ScheduleLog {
  id: number
  run_date: string
  report_date: string
  trigger: string
  status: 'sent' | 'failed'
  recipients: number
  error_message: string | null
  created_at: string
}
