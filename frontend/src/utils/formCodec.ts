// Konversi nilai form ↔ nilai API untuk isian bertipe tanggal, jam, huruf besar, dan teks kosong.
import type { FieldDef } from '@/types/form.types'

import { dayjs } from './time'

const DATE = 'YYYY-MM-DD'

/** Nilai form ↔ nilai API untuk jenis yang perlu diubah (tanggal, jam, huruf besar, teks kosong). */
export const fieldCodec = {
  toForm(fields: FieldDef[], data: Record<string, unknown>): Record<string, unknown> {
    const out = { ...data }
    for (const f of fields) {
      const v = data[f.name]
      if (v == null || v === '') continue
      if (f.type === 'date') out[f.name] = dayjs(String(v), DATE)
      if (f.type === 'time') out[f.name] = dayjs(String(v), 'HH:mm')
    }
    return out
  },
  toPayload(fields: FieldDef[], values: Record<string, unknown>): Record<string, unknown> {
    const out: Record<string, unknown> = {}
    for (const f of fields) {
      let v = values[f.name]
      if (v === undefined) continue
      if (dayjs.isDayjs(v)) v = f.type === 'time' ? v.format('HH:mm') : v.format(DATE)
      if (typeof v === 'string') {
        v = v.trim()
        if (f.type === 'upper' || f.type === 'combo') v = (v as string).toUpperCase()
        if (v === '') v = null
      }
      out[f.name] = v
    }
    return out
  },
}
