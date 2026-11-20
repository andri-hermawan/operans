import { useMemo } from 'react'

import { useAuthStore } from '@/stores/auth.store'
import type { MenuEntry, PageRoute } from '@/types/route.types'
import { canOpenPage } from '@/utils/permissions'
import { accessKeyOf, MENU } from '@/router/routes'
import { pruneMenu } from '@/utils/menu'

/** Menu yang boleh dilihat role ini (grup kosong disembunyikan). */
export function useVisibleMenu(): MenuEntry[] {
  const role = useAuthStore((s) => s.role)
  const access = useAuthStore((s) => s.access)
  const ready = useAuthStore((s) => s.accessReady)
  return useMemo(() => {
    if (!ready) return [] // hak akses belum dimuat: tidak menampilkan menu dulu agar tidak berkedip
    const ok = (r: PageRoute) => r.menu !== false && canOpenPage(role, r.resource, accessKeyOf(r), access)
    return pruneMenu(MENU, ok) // kelompok (di tingkat mana pun) yang seluruh isinya tak boleh dilihat role ini disembunyikan
  }, [role, access, ready])
}
