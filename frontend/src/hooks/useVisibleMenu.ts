import { useMemo } from 'react'

import { useAuthStore } from '@/stores/auth.store'
import type { MenuEntry, PageRoute } from '@/types/route.types'
import { can } from '@/utils/permissions'
import { MENU } from '@/router/routes'
import { pruneMenu } from '@/utils/menu'

/** Menu yang boleh dilihat role ini (grup kosong disembunyikan). */
export function useVisibleMenu(): MenuEntry[] {
  const role = useAuthStore((s) => s.role)
  return useMemo(() => {
    const ok = (r: PageRoute) => r.menu !== false && can(role, r.resource)
    return pruneMenu(MENU, ok) // kelompok (di tingkat mana pun) yang seluruh isinya tak boleh dilihat role ini disembunyikan
  }, [role])
}
