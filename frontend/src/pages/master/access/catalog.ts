// Daftar halaman & laporan yang bisa diatur di menu Hak Akses, dibaca dari router/routes.tsx
// (menambah halaman di sana otomatis muncul di sini) beserta logika menghitung hak per role / per user.
import type { AccessRule } from '@/types/access.types'
import type { ProjectRole, Resource } from '@/types/auth.types'
import type { MenuEntry, PageRoute } from '@/types/route.types'
import { isGroup } from '@/utils/menu'
import { can, isGrantable } from '@/utils/permissions'
import { accessKeyOf, MENU, ROUTES } from '@/router/routes'

export interface CatalogItem {
  key: string
  title: string
  resource: Resource
}

export interface CatalogGroup {
  title: string
  items: CatalogItem[]
}

export const ROLES: { value: ProjectRole; label: string }[] = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'dispatcher', label: 'Dispatcher' },
  { value: 'admin', label: 'Admin' },
]

const TOP_GROUP: Partial<Record<Resource, string>> = { monitoring: 'Monitoring', report: 'Report', documents: 'Dokumen SOP' }

function build(): CatalogGroup[] {
  const groups = new Map<string, CatalogItem[]>()
  const seen = new Set<string>()
  const add = (group: string, r: PageRoute) => {
    const key = accessKeyOf(r)
    if (key === false || key === 'master/access' || seen.has(key)) return // dibuka khusus superadmin / tidak bisa dibatasi / sudah ada
    seen.add(key)
    groups.set(group, [...(groups.get(group) ?? []), { key, title: r.title, resource: r.resource }])
  }
  const walk = (entries: MenuEntry[], path: string[]) => {
    for (const e of entries) {
      if (isGroup(e)) walk(e.children, [...path, e.title])
      else add(path.length ? path.join(' › ') : (TOP_GROUP[e.resource] ?? 'Lainnya'), e)
    }
  }
  walk(MENU, [])
  // halaman di luar menu yang tetap punya akses sendiri: laporan dan grafik unit
  ROUTES.filter((r) => r.menu === false && (r.path.startsWith('/report/') || r.path === '/graphic')).forEach((r) =>
    add(r.path === '/graphic' ? 'Monitoring' : 'Report', r),
  )
  const order = ['Monitoring', 'Report', 'Dokumen SOP']
  return [...groups.entries()]
    .map(([title, items]) => ({ title, items }))
    .sort((a, b) => {
      const ia = order.indexOf(a.title)
      const ib = order.indexOf(b.title)
      return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib)
    })
}

export const CATALOG: CatalogGroup[] = build()

/** Aturan bawaan (tanpa pengaturan superadmin). */
export const defaultAllowed = (role: ProjectRole, item: CatalogItem): boolean => can(role, item.resource)

/** Halaman data master/organisasi hanya bisa dibatasi, tidak bisa dibukakan untuk role di bawah admin. */
export const canBeGranted = (item: CatalogItem): boolean => isGrantable(item.resource)

export interface RuleIndex {
  role: Record<string, Record<string, boolean>> // role → kunci → boleh
  user: Record<string, Record<string, boolean>> // user_id → kunci → boleh
}

export function indexRules(rules: AccessRule[]): RuleIndex {
  const idx: RuleIndex = { role: {}, user: {} }
  for (const r of rules) {
    if (r.subject_type === 'role' && r.role) (idx.role[r.role] ??= {})[r.page_key] = r.allowed
    if (r.subject_type === 'user' && r.user_id) (idx.user[r.user_id] ??= {})[r.page_key] = r.allowed
  }
  return idx
}

/** Hasil akhir untuk satu role pada satu halaman (bawaan, lalu aturan role). */
export function roleAllowed(role: ProjectRole, item: CatalogItem, roleRules: Record<string, boolean> | undefined): boolean {
  const set = roleRules?.[item.key]
  if (set === false) return false
  if (set === true) return canBeGranted(item) || defaultAllowed(role, item)
  return defaultAllowed(role, item)
}

export type UserChoice = 'inherit' | 'allow' | 'block'

/** Hasil akhir untuk satu user: aturan user menimpa role. */
export function userAllowed(
  role: ProjectRole,
  item: CatalogItem,
  roleRules: Record<string, boolean> | undefined,
  userChoice: UserChoice,
): boolean {
  if (userChoice === 'block') return false
  if (userChoice === 'allow') return canBeGranted(item) || defaultAllowed(role, item)
  return roleAllowed(role, item, roleRules)
}
