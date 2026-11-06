// Akses project satu user: project mana saja yang boleh dimasuki dan role-nya di tiap project (khusus superadmin).
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { App, Button, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, Typography } from 'antd'
import { FolderKey, Plus, Trash2 } from 'lucide-react'

import { listProjects, removeProjectMember, setProjectMemberRole } from '@/services/api/project.api'
import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { UserRow } from '@/types/user.types'

const ROLE_OPTIONS = [
  { value: 'admin', label: 'Admin' },
  { value: 'dispatcher', label: 'Dispatcher' },
  { value: 'viewer', label: 'Viewer' },
]

export default function ProjectAccessButton({ user }: { user: UserRow }) {
  const { message } = App.useApp()
  const qc = useQueryClient()
  const selfId = useAuthStore((s) => s.user?.id)
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [projectId, setProjectId] = useState<string>()
  const [role, setRole] = useState('viewer')
  const projects = useQuery({ queryKey: ['project-access', 'projects'], queryFn: listProjects, enabled: open })

  const isSelf = user.id === selfId // backend menolak mengubah akses akun sendiri (mencegah terkunci)
  const joined = new Set(user.memberships.map((m) => m.project_id))
  const choices = (projects.data ?? [])
    .filter((p) => p.is_active && !joined.has(p.id))
    .map((p) => ({ value: p.id, label: `${p.project_code} — ${p.project_name}` }))

  /** Jalankan perubahan, lalu muat ulang daftar user supaya akses terbaru tampil. */
  const run = async (action: () => Promise<unknown>, done: string) => {
    setBusy(true)
    try {
      await action()
      message.success(done)
      await qc.invalidateQueries({ predicate: (q) => q.queryKey[1] === 'users' })
    } catch (e) {
      message.error(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  const add = () =>
    run(async () => {
      await setProjectMemberRole(projectId!, user.email, role)
      setProjectId(undefined)
    }, 'Akses project ditambahkan')

  return (
    <>
      <Tooltip title="Akses project">
        <Button size="small" type="text" icon={<FolderKey size={15} />} onClick={() => setOpen(true)} />
      </Tooltip>
      <Modal open={open} title={`Akses project — ${user.name}`} footer={null} width={560} onCancel={() => setOpen(false)} destroyOnHidden>
        {user.is_superadmin && (
          <Typography.Paragraph type="secondary">Super Admin otomatis bisa membuka semua project; daftar di bawah hanya keanggotaan tambahan.</Typography.Paragraph>
        )}
        {isSelf && <Typography.Paragraph type="warning">Akses akun sendiri tidak bisa diubah di sini (agar tidak terkunci). Minta superadmin lain.</Typography.Paragraph>}
        <Table
          size="small"
          rowKey="project_id"
          pagination={false}
          dataSource={user.memberships}
          locale={{ emptyText: 'Belum jadi anggota project mana pun' }}
          columns={[
            { title: 'Project', render: (_, m) => <Typography.Text strong>{m.project_code} — {m.project_name}</Typography.Text> },
            {
              title: 'Role',
              width: 140,
              render: (_, m) =>
                isSelf ? (
                  <Tag>{m.role}</Tag>
                ) : (
                  <Select
                    size="small"
                    style={{ width: 120 }}
                    value={m.role}
                    options={ROLE_OPTIONS}
                    disabled={busy}
                    onChange={(r) => run(() => setProjectMemberRole(m.project_id, user.email, r), 'Role diubah')}
                  />
                ),
            },
            {
              title: '',
              width: 48,
              render: (_, m) =>
                !isSelf && (
                  <Popconfirm
                    title={`Keluarkan dari ${m.project_code}?`}
                    okText="Keluarkan"
                    onConfirm={() => run(() => removeProjectMember(m.project_id, user.id), 'Akses dicabut')}
                  >
                    <Button size="small" type="text" danger disabled={busy} icon={<Trash2 size={14} />} />
                  </Popconfirm>
                ),
            },
          ]}
        />
        {!isSelf && (
          <Space.Compact style={{ width: '100%', marginTop: 16 }}>
            <Select
              style={{ flex: 1 }}
              placeholder={choices.length ? 'Tambah ke project…' : 'Semua project sudah ditambahkan'}
              loading={projects.isLoading}
              options={choices}
              value={projectId}
              onChange={setProjectId}
            />
            <Select style={{ width: 130 }} value={role} options={ROLE_OPTIONS} onChange={setRole} />
            <Button type="primary" icon={<Plus size={15} />} disabled={!projectId} loading={busy} onClick={add}>
              Tambah
            </Button>
          </Space.Compact>
        )}
      </Modal>
    </>
  )
}
