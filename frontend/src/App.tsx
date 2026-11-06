import { useEffect } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { App as AntApp, ConfigProvider, Spin } from 'antd'
import idID from 'antd/locale/id_ID'
import { RouterProvider } from 'react-router-dom'

import { queryClient } from '@/config/queryClient'
import { theme } from '@/config/theme'
import { useAuthStore } from '@/stores/auth.store'
import { router } from '@/router'

export default function App() {
  const status = useAuthStore((s) => s.status)
  const restore = useAuthStore((s) => s.restore)

  useEffect(() => {
    restore()
  }, [restore])

  return (
    <ConfigProvider theme={theme} locale={idID}>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          {status === 'restoring' ? <Spin size="large" fullscreen description="Memuat sesi..." /> : <RouterProvider router={router} />}
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  )
}
