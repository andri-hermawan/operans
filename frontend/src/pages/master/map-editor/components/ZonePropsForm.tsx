// Form properti satu area peta: isian sama dengan master (Category, Segment, HEX_Line, HEX_Fill, Priority).
import { AutoComplete, ColorPicker, Form, Input, InputNumber, Space } from 'antd'

import type { EditorLayer, ZoneProps } from '@/types/zoneEditor.types'
import { colorOfCategory } from '@/utils/zoneEditor'

interface Props {
  layer: EditorLayer
  /** Semua area di editor (untuk saran kategori & warna kategori yang sama). */
  layers: EditorLayer[]
  onChange: (p: Partial<ZoneProps>) => void
}

export default function ZonePropsForm({ layer, layers, onChange }: Props) {
  const p = layer.zoneProps
  const categories = [...new Set(layers.map((l) => l.zoneProps.Category.trim()).filter(Boolean))]

  /** Kategori sudah dipakai area lain → warnanya ikut (tetap bisa diganti lagi lewat pemilih warna). */
  const setCategory = (v: string) => {
    const same = colorOfCategory(layers, v, layer)
    onChange({ Category: v, ...same })
  }

  return (
    <Form layout="vertical" size="small">
      <div style={{ display: 'flex', gap: 12 }}>
        <Form.Item style={{ flex: 1 }} label="Category" required validateStatus={p.Category.trim() ? undefined : 'error'} help={p.Category.trim() ? undefined : 'Wajib diisi'}>
          <AutoComplete value={p.Category} options={categories.map((c) => ({ value: c }))} onChange={setCategory} placeholder="mis. Emplacement" maxLength={50} />
        </Form.Item>
        <Form.Item style={{ flex: 1 }} label="Segment" required validateStatus={p.Segment.trim() ? undefined : 'error'} help={p.Segment.trim() ? undefined : 'Wajib diisi'}>
          <Input value={p.Segment} onChange={(e) => onChange({ Segment: e.target.value })} placeholder="mis. Office" maxLength={100} />
        </Form.Item>
      </div>
      <Space size="large" align="start">
        <Form.Item label="Warna garis (HEX_Line)">
          <ColorPicker value={p.HEX_Line} disabledAlpha showText onChange={(c) => onChange({ HEX_Line: c.toHexString().toUpperCase() })} />
        </Form.Item>
        <Form.Item label="Warna isi (HEX_Fill)">
          <ColorPicker value={p.HEX_Fill} disabledAlpha showText onChange={(c) => onChange({ HEX_Fill: c.toHexString().toUpperCase() })} />
        </Form.Item>
      </Space>
      <Space size="large" align="start" wrap>
        <Form.Item label="BUFF_DIST">
          <InputNumber value={p.BUFF_DIST} onChange={(v) => onChange({ BUFF_DIST: v ?? 0 })} style={{ width: 110 }} />
        </Form.Item>
        <Form.Item label="ORIG_FID">
          <InputNumber value={p.ORIG_FID} precision={0} onChange={(v) => onChange({ ORIG_FID: v ?? 0 })} style={{ width: 110 }} />
        </Form.Item>
      </Space>
      <Space size="large" align="start" wrap>
        <Form.Item label="CODE">
          <Input value={p.CODE} onChange={(e) => onChange({ CODE: e.target.value })} maxLength={20} style={{ width: 110 }} />
        </Form.Item>
        <Form.Item label="CODE_2">
          <InputNumber value={p.CODE_2} onChange={(v) => onChange({ CODE_2: v ?? 0 })} style={{ width: 110 }} />
        </Form.Item>
      </Space>
      <Form.Item label="Priority" extra="Area bertumpuk: angka lebih tinggi yang dipakai (−100 sampai 100).">
        <InputNumber value={p.Priority} min={-100} max={100} precision={0} onChange={(v) => onChange({ Priority: v ?? 0 })} />
      </Form.Item>
    </Form>
  )
}
