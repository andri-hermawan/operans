import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useAuthStore } from '@/stores/auth.store'

/** Halaman di dalam aplikasi: wajib login. */
export function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status)
  const location = useLocation()
  if (status === 'signedOut') return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  return children
}
