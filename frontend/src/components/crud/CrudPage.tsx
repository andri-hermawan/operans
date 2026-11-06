// Halaman daftar + tambah/ubah/hapus + upload Excel + ekspor dari SATU konfigurasi.
// Dipakai halaman master & upload (pengganti pola 3 file × 13 halaman di FMS lama).
import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { App, Button, Dropdown, Form, Space, Tooltip, type MenuProps } from 'antd'
import { ChevronDown, Pencil, Plus, Trash2, Upload as UploadIcon } from 'lucide-react'

import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { CrudConfig, CrudCtx, Payload } from '@/types/crud.types'
import { useConfirm } from '@/hooks/useConfirm'
import { useFilters } from '@/hooks/useFilters'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import { useShifts } from '@/hooks/useShifts'
import { fieldCodec } from '@/utils/formCodec'
import { can } from '@/utils/permissions'
import FilterBar from '@/components/filters/FilterBar'
import FormFields from '@/components/form/FormFields'
import DataTable from '@/components/table/DataTable'
import FormDrawer from '@/components/ui/FormDrawer'
import PageHeader from '@/components/ui/PageHeader'

import ImportModal from './ImportModal'

export default function CrudPage<T>({ config: c }: { config: CrudConfig<T> }) {
  const { message } = App.useApp()
  const qc = useQueryClient()
  const confirm = useConfirm()
  const role = useAuthStore((s) => s.role)
  const projectId = useAuthStore((s) => s.project?.project_id) ?? ''
  const { data: shifts = [] } = useShifts()
  const now = useOperationalDefaults()
  const [filters, setFilters] = useFilters(c.defaultFilters ? c.defaultFilters(now) : {})
  const [form] = Form.useForm()
  const [editing, setEditing] = useState<{ mode: 'create' | 'edit'; row?: T; values: Payload; id: number }>()
  const [importOpen, setImportOpen] = useState(false)

  const ctx: CrudCtx = useMemo(
    () => ({ projectId, shifts, role, shiftId: (code) => shifts.find((s) => s.shift_code === code)?.id }),
    [projectId, shifts, role],
  )

  const allowed = {
    create: !!c.form?.create && can(role, c.resource, 'create') && (!c.createRoles || (!!role && c.createRoles.includes(role))),
    update: !!c.form?.update && can(role, c.resource, 'update'),
    delete: !!c.remove && can(role, c.resource, 'delete'),
    import: !!c.importType && can(role, c.resource, 'import'),
  }

  // shift dibutuhkan untuk mengubah kode shift → id sebelum memanggil API
  const needsShifts = c.filters?.some((f) => f.type === 'shift') ?? false
  const query = useQuery({
    queryKey: [projectId, c.queryKey, filters],
    enabled: !!projectId && (!needsShifts || shifts.length > 0) && (!c.defaultFilters || now.ready),
    queryFn: () => c.list(filters, ctx),
  })

  const refresh = () => qc.invalidateQueries({ queryKey: [projectId, c.queryKey] })

  const save = useMutation({
    mutationFn: async (values: Payload) => {
      const payload = fieldCodec.toPayload(c.form!.fields, values)
      return editing?.mode === 'edit' ? c.form!.update!(editing.row as T, payload, ctx) : c.form!.create!(payload, ctx)
    },
    onSuccess: () => {
      message.success('Data disimpan')
      setEditing(undefined)
      refresh()
    },
    onError: (e) => message.error(errorMessage(e)),
  })

  const open = (mode: 'create' | 'edit', row?: T) => {
    const fields = c.form!.fields
    const initial: Payload = {}
    fields.forEach((f) => f.initial !== undefined && (initial[f.name] = f.initial))
    const data = mode === 'edit' ? (c.form!.toForm ? c.form!.toForm(row as T) : (row as Payload)) : { ...initial, ...c.form!.initial?.(ctx, filters) }
    // nilai awal diberikan lewat initialValues (Form dibuat ulang tiap dibuka) — mengisi lewat setFieldsValue sebelum
    // laci terpasang tidak tersimpan
    setEditing({ mode, row, values: fieldCodec.toForm(fields, data), id: Date.now() })
  }

  const menu: MenuProps['items'] = [
    ...(allowed.create ? [{ key: 'add', icon: <Plus size={14} />, label: 'Add', onClick: () => open('create') }] : []),
    ...(allowed.import ? [{ key: 'import', icon: <UploadIcon size={14} />, label: 'Upload Excel', onClick: () => setImportOpen(true) }] : []),
  ]

  const actions =
    allowed.update || allowed.delete || c.rowActions
      ? (row: T) => (
          <Space size={4}>
            {allowed.update && (!c.form?.canEdit || c.form.canEdit(row, ctx)) && (
              <Tooltip title="Edit"><Button size="small" type="text" icon={<Pencil size={15} />} onClick={() => open('edit', row)} /></Tooltip>
            )}
            {allowed.delete && (
              <Tooltip title="Hapus">
                <Button
                  size="small"
                  type="text"
                  danger
                  icon={<Trash2 size={15} />}
                  onClick={() => confirm({ title: `Hapus ${c.remove!.label(row)}?`, action: () => c.remove!.action(row, ctx).then(refresh), success: 'Data dihapus' })}
                />
              </Tooltip>
            )}
            {c.rowActions?.(row, ctx)}
          </Space>
        )
      : undefined

  return (
    <>
      <PageHeader
        title={c.title}
        subtitle={c.subtitle}
        extra={
          menu.length > 0 && (
            <Dropdown menu={{ items: menu }} trigger={['click']}>
              <Button type="primary">Actions <ChevronDown size={14} /></Button>
            </Dropdown>
          )
        }
      />
      <DataTable<T>
        rows={query.data}
        columns={c.columns}
        rowKey={c.rowKey}
        loading={query.isFetching}
        error={query.error}
        onRetry={() => query.refetch()}
        searchable={c.searchable ?? true}
        exportName={c.exportName}
        actions={actions}
        toolbar={
          <>
            {c.filters && <FilterBar fields={c.filters} values={filters} onChange={setFilters} />}
            {c.summary && query.data && c.summary(query.data)}
          </>
        }
      />

      {c.form && (
        <FormDrawer
          open={!!editing}
          title={`${editing?.mode === 'edit' ? 'Edit' : 'Tambah'} ${c.title}`}
          width={c.form.width}
          onClose={() => setEditing(undefined)}
          onSubmit={() => form.submit()}
          saving={save.isPending}
        >
          {/* preserve={false}: nilai dibuang saat laci ditutup, supaya Tambah tidak membawa isi Edit sebelumnya */}
          <Form key={editing?.id} form={form} preserve={false} initialValues={editing?.values} layout="vertical" onFinish={(v) => save.mutate(v)} requiredMark>
            {editing && <FormFields fields={c.form.fields} mode={editing.mode} />}
          </Form>
        </FormDrawer>
      )}

      {c.importType && (
        <ImportModal type={c.importType} title={c.title} open={importOpen} onClose={() => setImportOpen(false)} onSaved={refresh} />
      )}
    </>
  )
}
