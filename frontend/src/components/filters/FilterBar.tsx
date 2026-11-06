import { useEffect, useState, type ReactNode } from 'react'
import { Badge, Button, DatePicker, Drawer, Grid, Input, Select, Space } from 'antd'
import { SlidersHorizontal } from 'lucide-react'

import type { FilterField, FilterValues } from '@/types/filter.types'
import { useEquipmentList } from '@/hooks/useEquipments'
import { useShifts } from '@/hooks/useShifts'
import { dayjs } from '@/utils/time'
import styles from '@/components/components.module.css'

interface Props {
  fields: FilterField[]
  values: FilterValues
  onChange: (patch: FilterValues) => void
  /** Tambahan di ujung baris (tombol lain). */
  extra?: ReactNode
}

const DATE = 'YYYY-MM-DD'
const ALL = '__all'

function Field({ f, values, onChange, block }: { f: FilterField; values: FilterValues; onChange: Props['onChange']; block: boolean }) {
  const { data: shifts } = useShifts()
  const needsEquipment = f.type === 'equipment'
  const { data: equipment } = useEquipmentList(false)
  const width = block ? '100%' : undefined
  const [search, setSearch] = useState(f.type === 'search' ? (values[f.key ?? 'q'] ?? '') : '')
  const searchKey = f.type === 'search' ? (f.key ?? 'q') : ''

  // pencarian dikirim 400 ms setelah berhenti mengetik
  useEffect(() => {
    if (f.type !== 'search') return
    const t = setTimeout(() => {
      if ((values[searchKey] ?? '') !== search) onChange({ [searchKey]: search || undefined })
    }, 400)
    return () => clearTimeout(t)
  }, [search, f.type, searchKey, values, onChange])

  switch (f.type) {
    case 'date': {
      const key = f.key ?? 'date'
      return (
        <DatePicker
          style={{ width: width ?? 150 }}
          format="DD/MM/YYYY"
          allowClear={f.allowClear ?? false}
          value={values[key] ? dayjs(values[key]) : null}
          onChange={(d) => onChange({ [key]: d ? d.format(DATE) : undefined })}
        />
      )
    }
    case 'dateRange': {
      const from = f.from ?? 'from'
      const to = f.to ?? 'to'
      return (
        <DatePicker.RangePicker
          style={{ width: width ?? 260 }}
          format="DD/MM/YYYY"
          allowClear={false}
          value={values[from] && values[to] ? [dayjs(values[from]), dayjs(values[to])] : null}
          disabledDate={f.maxDays ? (d, { from: start }) => !!start && Math.abs(d.diff(start, 'day')) >= f.maxDays! : undefined}
          onChange={(r) => onChange({ [from]: r?.[0]?.format(DATE), [to]: r?.[1]?.format(DATE) })}
        />
      )
    }
    case 'shift': {
      const key = f.key ?? 'shift'
      const options = (shifts ?? []).map((s) => ({ value: s.shift_code, label: `${s.shift_name} (${s.start_time}–${s.end_time})` }))
      return (
        <Select
          style={{ width: width ?? 190 }}
          value={values[key] ?? (f.allowAll ? ALL : undefined)}
          placeholder="Shift"
          options={f.allowAll ? [{ value: ALL, label: 'Semua shift' }, ...options] : options}
          onChange={(v: string) => onChange({ [key]: v === ALL ? undefined : v })}
        />
      )
    }
    case 'equipment': {
      const key = f.key ?? 'equipment'
      const options = needsEquipment ? (equipment ?? []).map((e) => ({ value: e.id, label: e.equipment_code })) : []
      return (
        <Select
          style={{ width: width ?? 200 }}
          showSearch={{ optionFilterProp: 'label' }}
          allowClear
          mode={f.multiple ? 'multiple' : undefined}
          maxTagCount="responsive"
          placeholder="Semua unit"
          options={options}
          value={f.multiple ? (values[key]?.split(',') ?? []) : values[key]}
          onChange={(v: string | string[] | undefined) => onChange({ [key]: Array.isArray(v) ? v.join(',') || undefined : v })}
        />
      )
    }
    case 'search':
      return <Input.Search allowClear style={{ width: width ?? 220 }} placeholder={f.placeholder ?? 'Cari...'} value={search} onChange={(e) => setSearch(e.target.value)} />
    case 'select':
      return (
        <Select
          style={{ width: width ?? 200 }}
          mode={f.multiple ? 'multiple' : undefined}
          maxTagCount="responsive"
          value={f.multiple ? (values[f.key]?.split(',') ?? []) : (values[f.key] ?? (f.allowAll ? ALL : undefined))}
          options={f.allowAll && !f.multiple ? [{ value: ALL, label: 'Semua' }, ...f.options] : f.options}
          placeholder={f.label}
          onChange={(v: string | string[]) => onChange({ [f.key]: Array.isArray(v) ? v.join(',') || undefined : v === ALL ? undefined : v })}
        />
      )
  }
}

const labelOf = (f: FilterField) =>
  f.label ?? { date: 'Tanggal', dateRange: 'Periode', shift: 'Shift', equipment: 'Unit', search: 'Cari', select: '' }[f.type]

/** Baris filter standar. Di HP berubah menjadi tombol "Filter" yang membuka laci. */
export default function FilterBar({ fields, values, onChange, extra }: Props) {
  const screens = Grid.useBreakpoint()
  const mobile = screens.md === false
  const [open, setOpen] = useState(false)

  const list = (block: boolean) =>
    fields.map((f, i) => (
      <div key={i} className={styles.filterField} style={block ? { width: '100%', flexDirection: 'column', alignItems: 'stretch', gap: 4 } : undefined}>
        <span className={styles.filterLabel}>{labelOf(f)}</span>
        <Field f={f} values={values} onChange={onChange} block={block} />
      </div>
    ))

  if (!mobile) {
    return (
      <div className={styles.filterBar}>
        {list(false)}
        {extra}
      </div>
    )
  }

  const active = fields.filter((f) => f.type !== 'date' && f.type !== 'dateRange' && values[(f as { key?: string }).key ?? f.type]).length
  return (
    <Space wrap>
      <Badge count={active} size="small">
        <Button icon={<SlidersHorizontal size={16} />} onClick={() => setOpen(true)}>Filter</Button>
      </Badge>
      {extra}
      <Drawer title="Filter" placement="bottom" size="auto" open={open} onClose={() => setOpen(false)} extra={<Button type="primary" onClick={() => setOpen(false)}>Terapkan</Button>}>
        <Space orientation="vertical" style={{ width: '100%' }} size={12}>{list(true)}</Space>
      </Drawer>
    </Space>
  )
}
