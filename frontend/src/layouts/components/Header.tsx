import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Avatar, Button, Dropdown, Grid, Layout, message, Select, Space, Typography, type MenuProps } from 'antd'
import { KeyRound, LogOut, Menu as MenuIcon, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { listProjects } from '@/services/api/project.api'
import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import { useUiStore } from '@/stores/ui.store'
import styles from '@/layouts/layout.module.css'

import AlertBell from './AlertBell'

const ROLE_LABEL: Record<string, string> = { superadmin: 'Super Admin', admin: 'Admin', dispatcher: 'Dispatcher', viewer: 'Viewer' }

/** Pilihan project aktif — hanya tampil bila ada lebih dari satu pilihan (superadmin: semua project aktif). */
function ProjectSwitcher({ compact }: { compact: boolean }) {
  const qc = useQueryClient()
  const project = useAuthStore((s) => s.project)
  const memberships = useAuthStore((s) => s.memberships)
  const switchProject = useAuthStore((s) => s.switchProject)
  const isSuperadmin = useAuthStore((s) => s.role) === 'superadmin'
  const all = useQuery({ queryKey: ['header', 'projects'], queryFn: listProjects, enabled: isSuperadmin, staleTime: 5 * 60_000 })
  // superadmin boleh membuka project mana pun (tanpa harus jadi anggota); selain itu hanya project keanggotaannya
  const choices = isSuperadmin
    ? (all.data ?? []).filter((p) => p.is_active).map((p) => ({ project_id: p.id, project_code: p.project_code, project_name: p.project_name }))
    : memberships
  if (choices.length < 2) {
    return project ? <Typography.Text strong className={styles.projectName}>{project.project_code} — {project.project_name}</Typography.Text> : null
  }
  return (
    <Select
      value={project?.project_id}
      placeholder="Pilih project"
      style={{ width: compact ? 140 : 260 }}
      options={choices.map((m) => ({ value: m.project_id, label: compact ? m.project_code : `${m.project_code} — ${m.project_name}` }))}
      onChange={async (id: string) => {
        try {
          await switchProject(id)
          qc.clear() // semua data milik project lama dibuang
          message.success('Project diganti')
        } catch (e) {
          message.error(errorMessage(e))
        }
      }}
    />
  )
}

export default function Header({ isMobile }: { isMobile: boolean }) {
  const navigate = useNavigate()
  const screens = Grid.useBreakpoint()
  const collapsed = useUiStore((s) => s.siderCollapsed)
  const toggle = useUiStore((s) => s.toggleSider)
  const user = useAuthStore((s) => s.user)
  const role = useAuthStore((s) => s.role)
  const logout = useAuthStore((s) => s.logout)

  const userMenu: MenuProps['items'] = [
    { key: 'profile', icon: <KeyRound size={16} />, label: 'Profil & Password', onClick: () => navigate('/profile') },
    { type: 'divider' },
    { key: 'logout', icon: <LogOut size={16} />, label: 'Keluar', danger: true, onClick: async () => { await logout(); navigate('/login') } },
  ]

  const ToggleIcon = isMobile ? MenuIcon : collapsed ? PanelLeftOpen : PanelLeftClose

  return (
    <Layout.Header className={styles.header}>
      <Space size={12}>
        <Button type="text" aria-label="Buka/tutup menu" icon={<ToggleIcon size={20} />} onClick={toggle} />
        <ProjectSwitcher compact={!screens.md} />
      </Space>
      <Space size={screens.sm ? 16 : 8}>
      <AlertBell />
      <Dropdown menu={{ items: userMenu }} placement="bottomRight" arrow trigger={['click']}>
        <button type="button" className={styles.userButton}>
          <Avatar size={34} className={styles.avatar}>{user?.name?.charAt(0).toUpperCase()}</Avatar>
          {screens.sm && (
            <span className={styles.userText}>
              <span className={styles.userName}>{user?.name}</span>
              <span className={styles.userRole}>{role ? ROLE_LABEL[role] : ''}</span>
            </span>
          )}
        </button>
      </Dropdown>
      </Space>
    </Layout.Header>
  )
}
