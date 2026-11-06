// System Data → User.
import { Tag } from 'antd'

import { useAuthStore } from '@/stores/auth.store'
import { setProjectMemberRole } from '@/services/api/project.api'
import { createUser, listUsers, updateUser } from '@/services/api/user.api'
import type { CrudConfig } from '@/types/crud.types'
import type { UserRow } from '@/types/user.types'
import CrudPage from '@/components/crud/CrudPage'
import { col } from '@/components/table/columns'

import ProjectAccessButton from './components/ProjectAccessButton'
import ResetPasswordButton from './components/ResetPasswordButton'

const ROLE_OPTIONS = [
  { value: 'admin', label: 'Admin' },
  { value: 'dispatcher', label: 'Dispatcher' },
  { value: 'viewer', label: 'Viewer' },
]
const ROLE_COLOR: Record<string, string> = { superadmin: 'red', admin: 'blue', dispatcher: 'geekblue', viewer: 'default' }

/** Backend menolak mengubah status aktif / superadmin akun sendiri (dan bagi non-superadmin). */
const isSelf = (v: Record<string, unknown>) => v.id === useAuthStore.getState().user?.id

const userConfig: CrudConfig<UserRow> = {
  title: 'User',
  resource: 'master',
  rowKey: 'id',
  queryKey: 'users',
  list: async (_f, ctx) =>
    // superadmin melihat SEMUA user (agar bisa menambahkan user ke project lain); admin hanya anggota project aktifnya
    (await listUsers(ctx.role === 'superadmin' ? undefined : ctx.projectId)).map((u) => ({ ...u, role: u.is_superadmin ? 'superadmin' : u.memberships.find((m) => m.project_id === ctx.projectId)?.role })),
  columns: [
    col.code('name', 'Name', { mobile: 'title' }),
    col.text('email', 'Email'),
    col.text('phone_number', 'Phone'),
    { key: 'role', title: 'Role', render: (r) => (r.role ? <Tag color={ROLE_COLOR[r.role]}>{r.role}</Tag> : '-') },
    col.active(),
    col.dateTime('last_login_at', 'Last Login'),
    col.dateTime('created_at', 'Created At', { mobile: 'hide' }),
  ],
  form: {
    fields: [
      { name: 'name', label: 'Nama', required: true },
      { name: 'email', label: 'Email', type: 'email', required: true, disabled: (_v, m) => m === 'edit' },
      { name: 'phone_number', label: 'No. HP (WhatsApp)', type: 'phone', help: 'Diperlukan untuk menerima notifikasi alert' },
      { name: 'password', label: 'Password', type: 'password', required: true, rules: [{ min: 8, message: 'Minimal 8 karakter' }], visible: (_v, m) => m === 'create' },
      { name: 'role', label: 'Role di project ini', type: 'select', options: ROLE_OPTIONS, required: true, initial: 'viewer', visible: (v) => !v.is_superadmin },
      { name: 'is_superadmin', label: 'Super Admin', type: 'switch', initial: false, roles: ['superadmin'], disabled: (v, m) => m === 'edit' && isSelf(v) },
      { name: 'is_active', label: 'Aktif', type: 'switch', initial: true, roles: ['superadmin'], visible: (_v, m) => m === 'edit', disabled: isSelf },
    ],
    create: (p, ctx) => createUser({ ...p, project_id: ctx.projectId, is_superadmin: !!p.is_superadmin, role: p.role ?? 'viewer' }),
    update: async (row, p, ctx) => {
      const upd: Record<string, unknown> = { name: p.name, phone_number: p.phone_number ?? '' }
      // status aktif & superadmin: hanya superadmin, dan tidak untuk akun sendiri (aturan backend)
      if (ctx.role === 'superadmin' && !isSelf(row as unknown as Record<string, unknown>)) {
        upd.is_active = p.is_active
        upd.is_superadmin = !!p.is_superadmin
      }
      await updateUser(row.id, upd)
      if (p.role && p.role !== row.role && !p.is_superadmin) await setProjectMemberRole(ctx.projectId, row.email, String(p.role))
    },
  },
  rowActions: (row, ctx) =>
    ctx.role === 'superadmin' ? (
      <>
        <ProjectAccessButton user={row} />
        <ResetPasswordButton user={row} />
      </>
    ) : null,
  exportName: 'user',
}

export default function UserPage() {
  return <CrudPage config={userConfig} />
}
