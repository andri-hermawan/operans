// Dokumen SOP — daftar, pencarian isi dokumen, dan pembaca per halaman (hanya baca).
// Hak akses dijalankan SERVER (project + role): halaman ini hanya menampilkan apa yang boleh dilihat user.
import { useMemo, useState } from 'react'
import { App, Button, Input, Select, Space, Tag } from 'antd'
import { BookOpen, FolderPlus, Plus, Printer, RefreshCw, Trash2, Users } from 'lucide-react'

import DataTable from '@/components/table/DataTable'
import { col } from '@/components/table/columns'
import PageHeader from '@/components/ui/PageHeader'
import SectionCard from '@/components/ui/SectionCard'
import { useAiDocuments } from '@/hooks/useAiDocuments'
import { useConfirm } from '@/hooks/useConfirm'
import { useDepartments } from '@/hooks/useDepartments'
import { useDocumentActions } from '@/hooks/useDocumentActions'
import { useFilters } from '@/hooks/useFilters'
import { useCan } from '@/hooks/usePermission'
import { usePrintDocument } from '@/hooks/usePrintDocument'
import type { AiDocument, DocumentFilters } from '@/types/aiDocument.types'
import type { Column } from '@/types/table.types'
import { READER_ROLE_LABEL, STATUS_LABEL } from '@/utils/documents'

import DepartmentModal from './components/DepartmentModal'
import DocumentReader, { type ReaderTarget } from './components/DocumentReader'
import DocumentRolesModal from './components/DocumentRolesModal'
import SearchResults from './components/SearchResults'
import UploadDocumentDrawer from './components/UploadDocumentDrawer'

export default function DocumentsPage() {
  const { message } = App.useApp()
  const confirm = useConfirm()
  const canManage = useCan('documents', 'create')
  const canPrint = useCan('documents', 'export')
  const print = usePrintDocument()
  const [f, setF] = useFilters()
  const { data: departments = [] } = useDepartments()
  const { deactivate, reprocess } = useDocumentActions()

  const filters = useMemo<DocumentFilters>(() => ({ department_id: f.department_id }), [f.department_id])
  const docs = useAiDocuments(filters)
  const query = f.q ?? ''
  const [text, setText] = useState(query)

  const [reading, setReading] = useState<ReaderTarget>()
  const [uploadOpen, setUploadOpen] = useState(false)
  const [deptOpen, setDeptOpen] = useState(false)
  const [rolesDoc, setRolesDoc] = useState<AiDocument>()

  const read = (d: AiDocument) => {
    if (d.processing_status !== 'ready') return message.warning(d.processing_error ?? 'Dokumen belum siap dibaca')
    setReading({ id: d.id, title: d.title, pageCount: d.page_count })
  }

  const columns: Column<AiDocument>[] = [
    col.code('title', 'Judul', { mobile: 'title' }),
    col.text('department_name', 'Department'),
    col.text('document_number', 'No. Dokumen'),
    col.text('revision', 'Revisi', { mobile: 'hide' }),
    col.number('page_count', 'Halaman'),
    col.date('created_at', 'Diunggah', { mobile: 'hide' }),
    ...(canManage
      ? [
          {
            key: 'processing_status',
            title: 'Status',
            sortable: true,
            render: (d) => (
              <Tag color={d.processing_status === 'ready' ? 'success' : d.processing_status === 'failed' ? 'error' : 'processing'} title={d.processing_error ?? undefined}>
                {STATUS_LABEL[d.processing_status]}
              </Tag>
            ),
            value: (d) => STATUS_LABEL[d.processing_status],
          } satisfies Column<AiDocument>,
          {
            key: 'roles',
            title: 'Boleh dibaca',
            render: (d) => (d.roles?.length ? d.roles.map((r) => <Tag key={r}>{READER_ROLE_LABEL[r]}</Tag>) : <Tag>Hanya admin</Tag>),
            value: (d) => (d.roles ?? []).map((r) => READER_ROLE_LABEL[r]).join(', ') || 'Hanya admin',
            mobile: 'hide',
          } satisfies Column<AiDocument>,
        ]
      : []),
  ]

  return (
    <>
      <PageHeader
        title="Dokumen SOP"
        subtitle="Prosedur kerja operasional. Hanya untuk dibaca; hak baca mengikuti project dan role Anda."
        extra={
          canManage && (
            <Space wrap>
              <Button icon={<FolderPlus size={14} />} onClick={() => setDeptOpen(true)}>Kelola Department</Button>
              <Button type="primary" icon={<Plus size={14} />} onClick={() => setUploadOpen(true)}>Tambah Dokumen</Button>
            </Space>
          )
        }
      />

      <SectionCard
        title="Cari di dalam dokumen"
        style={{ marginBottom: 16 }}
        extra={
          <Space wrap>
            <Select
              allowClear
              placeholder="Semua department"
              value={f.department_id}
              onChange={(v) => setF({ department_id: v })}
              options={departments.map((d) => ({ value: d.id, label: d.department_name }))}
              style={{ width: 200 }}
            />
          </Space>
        }
      >
        <Input.Search
          allowClear
          enterButton="Cari"
          size="large"
          maxLength={200}
          placeholder="Cari kata di dalam dokumen, mis. keselamatan kerja  (gunakan “frasa persis”, atau, -kecualikan)"
          value={text}
          onChange={(e) => {
            setText(e.target.value)
            if (!e.target.value) setF({ q: undefined })
          }}
          onSearch={(v) => setF({ q: v.trim() || undefined })}
        />
        {query && (
          <div style={{ marginTop: 12 }}>
            <SearchResults query={query} filters={filters} onOpen={(h) => setReading({ id: h.document_id, title: h.title, page: h.page_number })} />
          </div>
        )}
      </SectionCard>

      <SectionCard title="Daftar dokumen">
        <DataTable<AiDocument>
          rows={docs.data}
          columns={columns}
          rowKey="id"
          loading={docs.isLoading}
          error={docs.error}
          onRetry={() => docs.refetch()}
          searchable
          onRowClick={read}
          actions={(d) => (
            <Space size={4} onClick={(e) => e.stopPropagation()}>
              <Button size="small" icon={<BookOpen size={14} />} disabled={d.processing_status !== 'ready'} onClick={() => read(d)}>Baca</Button>
              {canPrint && (
                <Button size="small" icon={<Printer size={14} />} disabled={d.processing_status !== 'ready'} onClick={() => print(d.id)}>Cetak</Button>
              )}
              {canManage && (
                <>
                  {(
                    <Button
                      size="small"
                      icon={<RefreshCw size={14} />}
                      title="Proses ulang (baca ulang teks, termasuk OCR bila aktif)"
                      onClick={() =>
                        confirm({
                          title: `Proses ulang “${d.title}”?`,
                          content: 'Teks dibaca ulang dari file yang tersimpan (halaman scan atau lampiran bergambar dibaca dengan OCR bila diaktifkan di server). Selama diproses, dokumen tidak terlihat oleh pembaca.',
                          okText: 'Ya, Proses Ulang',
                          danger: false,
                          action: () => reprocess.mutateAsync(d.id),
                          success: 'Dokumen diproses ulang',
                        })
                      }
                    />
                  )}
                  <Button size="small" icon={<Users size={14} />} title="Atur hak baca" onClick={() => setRolesDoc(d)} />
                  <Button
                    size="small"
                    danger
                    icon={<Trash2 size={14} />}
                    title="Tarik dokumen"
                    onClick={() =>
                      confirm({
                        title: `Tarik dokumen “${d.title}”?`,
                        content: 'Dokumen hilang dari daftar dan pencarian semua pengguna. Riwayat tetap tersimpan.',
                        okText: 'Ya, Tarik',
                        action: () => deactivate.mutateAsync(d.id),
                        success: 'Dokumen ditarik',
                      })
                    }
                  />
                </>
              )}
            </Space>
          )}
        />
      </SectionCard>

      <DocumentReader target={reading} onClose={() => setReading(undefined)} />
      {canManage && (
        <>
          <UploadDocumentDrawer open={uploadOpen} onClose={() => setUploadOpen(false)} />
          <DepartmentModal open={deptOpen} onClose={() => setDeptOpen(false)} />
          <DocumentRolesModal doc={rolesDoc} onClose={() => setRolesDoc(undefined)} />
        </>
      )}
    </>
  )
}
