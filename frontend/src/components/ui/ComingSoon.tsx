import { Result } from 'antd'
import { Hammer } from 'lucide-react'

import { brand } from '@/utils/colors'

import PageHeader from './PageHeader'

const PHASE: Record<string, string> = {
  F2: 'F2 — Master & Upload',
  F3: 'F3 — Pemantauan (peta & riwayat)',
  F4: 'F4 — Alert & Report',
  F5: 'F5 — Jadwal laporan & cetak',
}

/** Penanda halaman yang sudah ada di menu tetapi belum dibuat. */
export default function ComingSoon({ title, phase }: { title: string; phase?: string }) {
  return (
    <>
      <PageHeader title={title} />
      <Result
        icon={<Hammer size={56} color={brand.primary} />}
        title="Halaman sedang dibuat"
        subTitle={phase ? `Dikerjakan pada tahap ${PHASE[phase] ?? phase}.` : undefined}
      />
    </>
  )
}
