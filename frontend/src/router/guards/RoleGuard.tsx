import type { ReactNode } from 'react'
import { Button, Result, Spin } from 'antd'
import { useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/stores/auth.store'
import type { PageRoute } from '@/types/route.types'
import { canOpenPage } from '@/utils/permissions'
import { accessKeyOf } from '@/router/routes'

/** Hak akses halaman: aturan bawaan (router/routes.tsx + utils/permissions.ts) dan pengaturan superadmin (Hak Akses). */
export function RequireAccess({ route, children }: { route: PageRoute; children: ReactNode }) {
  const role = useAuthStore((s) => s.role)
  const access = useAuthStore((s) => s.access)
  const ready = useAuthStore((s) => s.accessReady)
  const navigate = useNavigate()
  if (!ready) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!canOpenPage(role, route.resource, accessKeyOf(route), access)) {
    return (
      <Result
        status="403"
        title="Akses ditolak"
        subTitle="Role Anda tidak memiliki akses ke halaman ini."
        extra={<Button type="primary" onClick={() => navigate('/')}>Kembali</Button>}
      />
    )
  }
  return children
}
