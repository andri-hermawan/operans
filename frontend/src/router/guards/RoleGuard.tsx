import type { ReactNode } from 'react'
import { Button, Result } from 'antd'
import { useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/stores/auth.store'
import type { PageRoute } from '@/types/route.types'
import { can } from '@/utils/permissions'
import { HOME } from '@/router/routes'

/** Hak akses halaman dibaca dari router/routes.tsx. */
export function RequireAccess({ route, children }: { route: PageRoute; children: ReactNode }) {
  const role = useAuthStore((s) => s.role)
  const navigate = useNavigate()
  if (!can(role, route.resource)) {
    return (
      <Result
        status="403"
        title="Akses ditolak"
        subTitle="Role Anda tidak memiliki akses ke halaman ini."
        extra={<Button type="primary" onClick={() => navigate(HOME)}>Kembali</Button>}
      />
    )
  }
  return children
}
