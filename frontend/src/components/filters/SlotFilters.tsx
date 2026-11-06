// Filter standar panel halaman peta/riwayat: baris 1 = Tanggal + Shift, baris 2 = Asset ID (+ isian tambahan, mis. Alert).
// Label di kiri isian. Nilainya disimpan di URL lewat useSlotFilters (values/set dari sana).
import type { ReactNode } from 'react'
import { DatePicker, Select, Space } from 'antd'

import type { FilterValues } from '@/types/filter.types'
import { useEquipmentList } from '@/hooks/useEquipments'
import { useShifts } from '@/hooks/useShifts'
import { dayjs } from '@/utils/time'
import styles from '@/components/monitoring/monitoring.module.css'

const ALL = '__all'

interface Props {
  values: FilterValues
  onChange: (patch: FilterValues) => void
  /** Isian tambahan di samping Asset ID (mis. <SlotFilters.Select label="Alert" …/>). Tanpa ini, Asset ID memenuhi satu baris. */
  extra?: ReactNode
  /** false = tanpa Asset ID (halaman yang tidak memilih satu unit); `extra` lalu memenuhi satu baris. */
  asset?: boolean
}

export default function SlotFilters({ values, onChange, extra, asset = true }: Props) {
  const { data: shifts = [] } = useShifts()
  const { data: equipment } = useEquipmentList(false)
  return (
    <div className={styles.panelFilters} style={{ gridTemplateColumns: '1fr 1fr' }}>
      <Space.Compact block>
        <Space.Addon>Tanggal</Space.Addon>
        <DatePicker
          style={{ width: '100%' }}
          format="DD/MM/YYYY"
          allowClear={false}
          value={values.date ? dayjs(values.date) : null}
          onChange={(d) => d && onChange({ date: d.format('YYYY-MM-DD') })}
        />
      </Space.Compact>
      <Space.Compact block>
        <Space.Addon>Shift</Space.Addon>
        <Select
          style={{ width: '100%' }}
          value={values.shift ?? ALL}
          options={[{ value: ALL, label: 'Semua shift' }, ...shifts.map((x) => ({ value: x.shift_code, label: x.shift_name }))]}
          onChange={(v: string) => onChange({ shift: v === ALL ? undefined : v })}
        />
      </Space.Compact>
      {asset && (
        <Space.Compact block style={extra ? undefined : { gridColumn: '1 / -1' }}>
          <Space.Addon>Asset ID</Space.Addon>
          <Select
            showSearch={{ optionFilterProp: 'label' }}
            allowClear
            style={{ width: '100%' }}
            placeholder="Pilih unit"
            value={values.equipment}
            options={(equipment ?? []).map((e) => ({ value: e.id, label: e.equipment_code }))}
            onChange={(v?: string) => onChange({ equipment: v })}
          />
        </Space.Compact>
      )}
      {extra && <div style={asset ? undefined : { gridColumn: '1 / -1' }}>{extra}</div>}
    </div>
  )
}

/** Isian pilihan dengan label di kiri (untuk `extra`). `allLabel` = teks pilihan "semua". */
export function SlotSelect({ label, value, options, allLabel = 'Semua', onChange }: {
  label: string
  value?: string
  options: { value: string; label: string }[]
  allLabel?: string
  onChange: (v?: string) => void
}) {
  return (
    <Space.Compact block>
      <Space.Addon>{label}</Space.Addon>
      <Select
        style={{ width: '100%' }}
        value={value ?? ALL}
        options={[{ value: ALL, label: allLabel }, ...options]}
        onChange={(v: string) => onChange(v === ALL ? undefined : v)}
      />
    </Space.Compact>
  )
}
