import { useMemo } from 'react'
import { Drawer, Layout, Menu, type MenuProps } from 'antd'
import { useLocation, useNavigate } from 'react-router-dom'

import { LAYOUT } from '@/config/theme'
import { useUiStore } from '@/stores/ui.store'
import type { MenuEntry } from '@/types/route.types'
import { useVisibleMenu } from '@/hooks/useVisibleMenu'
import { groupKeysTo, isGroup } from '@/utils/menu'
import logo from '@/assets/logo.png'

import styles from '../layout.module.css'

type Item = Required<MenuProps>['items'][number]

const toItem = (e: MenuEntry): Item =>
  isGroup(e)
    ? { key: e.key, icon: e.icon, label: e.title, children: e.children.map(toItem) }
    : { key: e.path, icon: e.icon, label: e.title }

function Brand({ collapsed }: { collapsed: boolean }) {
  return (
    <div className={styles.brand} style={{ justifyContent: collapsed ? 'center' : 'flex-start' }}>
      <img src={logo} alt="RMKO" className={styles.brandLogo} />
      {!collapsed && <span className={styles.brandText}>OPERANS</span>}
    </div>
  )
}

export default function Sidebar({ isMobile }: { isMobile: boolean }) {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const collapsed = useUiStore((s) => s.siderCollapsed)
  const setCollapsed = useUiStore((s) => s.setSiderCollapsed)
  const menu = useVisibleMenu()

  const items = useMemo(() => menu.map(toItem), [menu])
  const openKeys = useMemo(() => groupKeysTo(menu, pathname), [menu, pathname])

  const content = (
    <>
      <Brand collapsed={collapsed && !isMobile} />
      <Menu
        mode="inline"
        className={styles.menu}
        selectedKeys={[pathname]}
        defaultOpenKeys={openKeys}
        items={items}
        onClick={({ key }) => {
          navigate(key)
          if (isMobile) setCollapsed(true)
        }}
      />
    </>
  )

  // HP: menu sebagai laci dari kiri, menutup sendiri setelah memilih halaman.
  if (isMobile) {
    return (
      <Drawer placement="left" open={!collapsed} onClose={() => setCollapsed(true)} size={LAYOUT.siderWidth} closable={false} styles={{ body: { padding: 0 } }}>
        {content}
      </Drawer>
    )
  }

  return (
    <Layout.Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      width={LAYOUT.siderWidth}
      collapsedWidth={LAYOUT.siderCollapsed}
      className={styles.sider}
    >
      {content}
    </Layout.Sider>
  )
}
