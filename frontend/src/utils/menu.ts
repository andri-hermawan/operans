// Logika menu samping (fungsi biasa, tanpa tampilan): kelompok bisa bersarang (Record / Upload → Transaction → halaman).
import type { MenuEntry, MenuGroup, PageRoute } from '@/types/route.types'

export const isGroup = (e: MenuEntry): e is MenuGroup => 'children' in e

/** Semua halaman di dalam sebuah entri, termasuk yang ada di kelompok bersarang. */
export const pagesOf = (e: MenuEntry): PageRoute[] => (isGroup(e) ? e.children.flatMap(pagesOf) : [e])

/** Buang halaman yang tidak boleh dilihat; kelompok (di tingkat mana pun) yang jadi kosong ikut dibuang. */
export const pruneMenu = (entries: MenuEntry[], visible: (page: PageRoute) => boolean): MenuEntry[] =>
  entries.flatMap<MenuEntry>((e) => {
    if (!isGroup(e)) return visible(e) ? [e] : []
    const children = pruneMenu(e.children, visible)
    return children.length ? [{ ...e, children }] : []
  })

/** Kunci semua kelompok (dari luar ke dalam) yang berisi halaman ini, agar kelompok itu terbuka saat halaman dibuka. */
export const groupKeysTo = (entries: MenuEntry[], path: string): string[] => {
  for (const e of entries) {
    if (!isGroup(e)) continue
    if (e.children.some((c) => !isGroup(c) && c.path === path)) return [e.key]
    const deeper = groupKeysTo(e.children, path)
    if (deeper.length) return [e.key, ...deeper]
  }
  return []
}
