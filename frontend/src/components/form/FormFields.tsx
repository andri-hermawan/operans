// Isian form dari konfigurasi. Halaman menulis daftar field (nama, jenis, wajib, sumber pilihan),
// komponen ini yang menggambar input, aturan validasi & tata letak (2 kolom di desktop, 1 kolom di HP).
import { AutoComplete, Col, DatePicker, Form, Input, InputNumber, Radio, Row, Select, Switch, TimePicker } from 'antd'
import type { Rule } from 'antd/es/form'

import { useAuthStore } from '@/stores/auth.store'
import type { FieldDef } from '@/types/form.types'
import { useOptions } from '@/hooks/useOptions'

// Form.Item menyuntikkan value/onChange (atau checked untuk switch) ke komponen ini → harus diteruskan ke input.
function Control({ f, disabled, ...inject }: { f: FieldDef; disabled: boolean } & Record<string, unknown>) {
  const { options, loading } = useOptions(f.options)
  const common = { disabled, placeholder: f.placeholder, style: { width: '100%' }, ...inject }
  switch (f.type ?? 'text') {
    case 'upper':
      return <Input {...common} style={{ ...common.style, textTransform: 'uppercase' }} />
    case 'email':
      return <Input {...common} type="email" autoComplete="off" />
    case 'phone':
      return <Input {...common} inputMode="tel" placeholder={f.placeholder ?? '62812xxxxxxx'} />
    case 'password':
      return <Input.Password {...common} autoComplete="new-password" />
    case 'textarea':
      return <Input.TextArea {...common} rows={3} />
    case 'number':
      return <InputNumber {...common} min={f.min} max={f.max} suffix={f.unit} decimalSeparator="," />
    case 'switch':
      return <Switch disabled={disabled} {...inject} />
    case 'radio':
      return <Radio.Group disabled={disabled} {...inject} options={options} optionType="button" buttonStyle="solid" />
    case 'select':
    case 'multiSelect':
      return (
        <Select
          {...common}
          mode={f.type === 'multiSelect' ? 'multiple' : undefined}
          loading={loading}
          options={options}
          allowClear={!f.required}
          showSearch={{ optionFilterProp: 'label' }}
        />
      )
    case 'combo': // pilih dari daftar ATAU ketik baru (mis. kode unit yang belum ada di master)
      return <AutoComplete {...common} options={options} filterOption={(input, o) => String(o?.value ?? '').toUpperCase().includes(input.toUpperCase())} />
    case 'date':
      return <DatePicker {...common} format="DD/MM/YYYY" />
    case 'time':
      return <TimePicker {...common} format="HH:mm" minuteStep={1} needConfirm={false} />
    default:
      return <Input {...common} />
  }
}

export default function FormFields({ fields, mode }: { fields: FieldDef[]; mode: 'create' | 'edit' }) {
  const form = Form.useFormInstance()
  const values = (Form.useWatch([], form) ?? {}) as Record<string, unknown>
  const role = useAuthStore((s) => s.role)
  return (
    <Row gutter={16}>
      {fields
        .filter((f) => (!f.roles || (!!role && f.roles.includes(role))) && (!f.visible || f.visible(values, mode)))
        .map((f) => {
          const rules: Rule[] = [...(f.required ? [{ required: true, message: `${f.label} wajib diisi` }] : []), ...(f.type === 'email' ? [{ type: 'email' as const, message: 'Format email tidak valid' }] : []), ...(f.rules ?? [])]
          return (
            <Col key={f.name} xs={24} md={f.span === 'full' || f.type === 'textarea' || f.type === 'multiSelect' ? 24 : 12}>
              <Form.Item name={f.name} label={f.label} rules={rules} help={f.help} valuePropName={f.type === 'switch' ? 'checked' : 'value'}>
                <Control f={f} disabled={!!f.disabled?.(values, mode)} />
              </Form.Item>
            </Col>
          )
        })}
    </Row>
  )
}
