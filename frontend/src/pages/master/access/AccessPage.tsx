// System Data → Hak Akses (khusus superadmin): halaman & laporan apa yang boleh dibuka tiap role / user, per project (ADR 0018).
import { useState } from 'react'
import { Alert, Select, Spin, Tabs } from 'antd'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { listAccessRules } from '@/services/api/access.api'
import { listProjects } from '@/services/api/project.api'
import { useAuthStore } from '@/stores/auth.store'
import PageHeader from '@/components/ui/PageHeader'

import styles from './access.module.css'
import RoleMatrix from './RoleMatrix'
import UserMatrix from './UserMatrix'

export default function AccessPage() {
  const qc = useQueryClient()
  const activeProject = useAuthStore((s) => s.project?.project_id)
  const projects = useQuery({ queryKey: ['access-projects'], queryFn: listProjects, staleTime: 5 * 60_000 })
  const active = (projects.data ?? []).filter((p) => p.is_active)
  const [picked, setPicked] = useState<string>()
  const projectId = picked ?? activeProject ?? active[0]?.id

  const rules = useQuery({
    queryKey: ['access-rules', projectId],
    queryFn: () => listAccessRules(projectId!),
    enabled: !!projectId,
  })
  const reload = () => qc.invalidateQueries({ queryKey: ['access-rules', projectId] })

  return (
    <>
      <PageHeader title="Hak Akses" subtitle="Halaman dan laporan yang boleh dibuka tiap role atau user, per project. Hanya superadmin." />
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        title="Pengaturan ini berlaku pada menu dan halaman. Perubahan diterima user setelah login ulang atau ganti project. Pemeriksaan di server (API) belum diterapkan."
      />
      <div className={styles.projectBar}>
        <b>Project</b>
        <Select
          style={{ minWidth: 300 }}
          loading={projects.isLoading}
          value={projectId}
          onChange={setPicked}
          options={active.map((p) => ({ value: p.id, label: `${p.project_code} — ${p.project_name}` }))}
        />
      </div>
      {!projectId || rules.isLoading ? (
        <Spin style={{ display: 'block', margin: '40px auto' }} />
      ) : rules.isError ? (
        <Alert type="error" showIcon title="Gagal memuat aturan hak akses" />
      ) : (
        <Tabs
          destroyOnHidden
          items={[
            { key: 'role', label: 'Per Role', children: <RoleMatrix key={projectId} projectId={projectId} rules={rules.data ?? []} onSaved={reload} /> },
            { key: 'user', label: 'Per User', children: <UserMatrix key={projectId} projectId={projectId} rules={rules.data ?? []} onSaved={reload} /> },
          ]}
        />
      )}
    </>
  )
}
