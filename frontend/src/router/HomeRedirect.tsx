import { Navigate } from 'react-router-dom'
import { Spin } from 'antd'

import { useVisibleMenu } from '@/hooks/useVisibleMenu'
import { useAuthStore } from '@/stores/auth.store'
import { pagesOf } from '@/utils/menu'

import { HOME } from './routes'

/** Halaman awal: HOME bila boleh dibuka, kalau tidak halaman pertama di menu yang boleh dibuka user ini. */
export function HomeRedirect() {
  const ready = useAuthStore((s) => s.accessReady)
  const pages = useVisibleMenu().flatMap(pagesOf)
  if (!ready) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  const target = pages.some((p) => p.path === HOME) ? HOME : (pages[0]?.path ?? '/profile')
  return <Navigate to={target} replace />
}
