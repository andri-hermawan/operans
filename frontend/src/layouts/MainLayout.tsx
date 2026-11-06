import { Suspense, useEffect } from 'react'
import { Grid, Layout, Spin } from 'antd'
import { Outlet, useLocation } from 'react-router-dom'

import { useUiStore } from '@/stores/ui.store'
import Header from '@/layouts/components/Header'
import Sidebar from '@/layouts/components/Sidebar'
import { routeOf } from '@/router/routes'

import styles from './layout.module.css'

export default function MainLayout() {
  const screens = Grid.useBreakpoint()
  const isMobile = screens.md === false
  const { pathname } = useLocation()
  const setCollapsed = useUiStore((s) => s.setSiderCollapsed)

  useEffect(() => {
    document.title = `${routeOf(pathname)?.title ?? 'OPERANS'} — OPERANS`
  }, [pathname])

  // Pindah ke ukuran HP: menu samping ditutup.
  useEffect(() => {
    if (isMobile) setCollapsed(true)
  }, [isMobile, setCollapsed])

  return (
    <Layout className={styles.shell}>
      <Sidebar isMobile={isMobile} />
      <Layout className={styles.main}>
        <Header isMobile={isMobile} />
        <Layout.Content className={styles.content}>
          <Suspense fallback={<Spin size="large" className={styles.pageSpin} />}>
            <Outlet />
          </Suspense>
        </Layout.Content>
      </Layout>
    </Layout>
  )
}
