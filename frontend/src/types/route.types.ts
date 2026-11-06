// Daftar halaman (menu, judul, hak akses). Datanya ada di router/routes.tsx.
import type { ComponentType, LazyExoticComponent, ReactNode } from 'react'

import type { Resource } from '@/types/auth.types'

export type Page = LazyExoticComponent<ComponentType>

export interface PageRoute {
  path: string
  title: string
  icon?: ReactNode
  resource: Resource
  /** Halaman yang belum dibuat: tampilkan "segera hadir" + tahapnya. */
  phase?: string
  page?: Page
  /** false = tidak tampil di menu (mis. halaman detail). */
  menu?: boolean
}

export interface MenuGroup {
  key: string
  title: string
  icon: ReactNode
  /** Halaman, atau kelompok lagi (menu bersarang: Record / Upload → Transaction → halaman). */
  children: MenuEntry[]
}

export type MenuEntry = PageRoute | MenuGroup
