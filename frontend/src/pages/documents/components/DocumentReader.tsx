// Pembaca dokumen: satu halaman per layar, hanya baca. Dua tampilan: "Teks" (hasil ekstraksi) dan "Asli" (gambar halaman
// bertanda air pembaca, supaya tabel/diagram/tata letak tampil seperti di PDF-nya). Tombol "Cetak PDF" hanya untuk superadmin.
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Drawer, Grid, InputNumber, Segmented, Skeleton, Space, Typography } from 'antd'
import { ChevronLeft, ChevronRight, Printer } from 'lucide-react'

import { useDocumentPage, useDocumentPageImage } from '@/hooks/useAiDocuments'
import { useCan } from '@/hooks/usePermission'
import { usePrintDocument } from '@/hooks/usePrintDocument'
import { errorMessage } from '@/services/http/ApiError'
import { brand } from '@/utils/colors'

export interface ReaderTarget {
  id: string
  title: string
  /** Halaman yang dibuka pertama kali (default 1). */
  page?: number
  /** Jumlah halaman bila sudah diketahui (daftar dokumen); bila tidak, diambil dari jawaban server. */
  pageCount?: number | null
}

interface Props {
  target: ReaderTarget | undefined
  onClose: () => void
}

type Mode = 'text' | 'image'

export default function DocumentReader({ target, onClose }: Props) {
  const screens = Grid.useBreakpoint()
  const canPrint = useCan('documents', 'export')
  const print = usePrintDocument()
  return (
    <Drawer
      open={!!target}
      onClose={onClose}
      title={target?.title}
      size={screens.md === false ? '100%' : 760}
      destroyOnHidden
      extra={canPrint && target && <Button icon={<Printer size={14} />} onClick={() => print(target.id)}>Cetak PDF</Button>}
    >
      {/* key: membuka dokumen/halaman lain = isi baru yang mulai dari halaman yang diminta */}
      {target && <ReaderBody key={`${target.id}:${target.page ?? 1}`} target={target} />}
    </Drawer>
  )
}

function ReaderBody({ target }: { target: ReaderTarget }) {
  const [page, setPage] = useState(target.page ?? 1)
  const [mode, setMode] = useState<Mode>('text')
  const text = useDocumentPage(target.id, page)
  const image = useDocumentPageImage(target.id, page, mode === 'image')
  const pageCount = text.data?.page_count ?? target.pageCount ?? undefined
  const go = (n: number) => setPage(Math.max(1, Math.min(n, pageCount ?? n)))

  // gambar dari server (butuh token) → alamat sementara di browser; dilepas saat halaman berganti/ditutup
  const imageUrl = useMemo(() => (image.data ? URL.createObjectURL(image.data) : undefined), [image.data])
  useEffect(() => () => { if (imageUrl) URL.revokeObjectURL(imageUrl) }, [imageUrl])

  const active = mode === 'text' ? text : image
  return (
    <>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }} wrap>
        <Segmented<Mode> value={mode} onChange={setMode} options={[{ value: 'text', label: 'Teks' }, { value: 'image', label: 'Asli (gambar)' }]} />
        <Space>
          Halaman
          <InputNumber min={1} max={pageCount} value={page} onChange={(v) => v && go(v)} style={{ width: 72 }} />
          {pageCount && <>dari {pageCount}</>}
        </Space>
      </Space>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Button icon={<ChevronLeft size={16} />} disabled={page <= 1} onClick={() => go(page - 1)}>Sebelumnya</Button>
        <Button disabled={!!pageCount && page >= pageCount} onClick={() => go(page + 1)}>
          Berikutnya <ChevronRight size={16} style={{ verticalAlign: 'middle' }} />
        </Button>
      </Space>

      {active.error ? (
        <Alert type="error" showIcon title={errorMessage(active.error)} />
      ) : mode === 'text' ? (
        !text.data && text.isFetching ? (
          <Skeleton active paragraph={{ rows: 12 }} />
        ) : (
          <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', lineHeight: 1.7, fontSize: 15 }}>
            {text.data?.content || <Typography.Text type="secondary">(halaman ini tidak berisi teks)</Typography.Text>}
          </Typography.Paragraph>
        )
      ) : !imageUrl ? (
        <Skeleton.Image active style={{ width: '100%', height: 360 }} />
      ) : (
        <img src={imageUrl} alt={`${target.title}, halaman ${page}`} style={{ width: '100%', border: `1px solid ${brand.border}`, borderRadius: 8 }} />
      )}
      <Typography.Paragraph style={{ color: brand.muted, fontSize: 12, marginTop: 12 }}>
        Dokumen hanya untuk dibaca di sini. Setiap pembukaan halaman tercatat; tampilan gambar memuat nama pembaca dan waktu.
      </Typography.Paragraph>
    </>
  )
}
