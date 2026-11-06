// Status login. Access token hanya di memori; refresh token di localStorage agar sesi bertahan saat halaman
// dimuat ulang (rencana produksi: cookie httpOnly — ADR 0010 bagian 4).
import { create } from 'zustand'

import { authApi } from '@/services/api/auth.api'
import { setAuthBridge } from '@/services/http/client'
import type { AuthStatus, Membership, Role, Session, SessionUser } from '@/types/auth.types'

const REFRESH_KEY = 'operans.refresh'
const storage = {
  get: () => {
    try {
      return localStorage.getItem(REFRESH_KEY) ?? undefined
    } catch {
      return undefined
    }
  },
  set: (v?: string) => {
    try {
      if (v) localStorage.setItem(REFRESH_KEY, v)
      else localStorage.removeItem(REFRESH_KEY)
    } catch {
      /* penyimpanan diblokir: sesi hanya bertahan selama tab terbuka */
    }
  },
}

interface AuthState {
  status: AuthStatus
  accessToken?: string
  refreshToken?: string
  user?: SessionUser
  project: Membership | null
  memberships: Membership[]
  /** Role efektif: superadmin, atau role di project aktif. */
  role?: Role
  login: (email: string, password: string) => Promise<void>
  restore: () => Promise<void>
  refresh: () => Promise<boolean>
  switchProject: (projectId: string) => Promise<void>
  logout: () => Promise<void>
}

let restoring: Promise<void> | null = null

const roleOf = (s: Session): Role | undefined => (s.user.is_superadmin ? 'superadmin' : s.project?.role)

export const useAuthStore = create<AuthState>((set, get) => {
  const apply = (s: Session) => {
    storage.set(s.refresh_token)
    set({ status: 'signedIn', accessToken: s.access_token, refreshToken: s.refresh_token, user: s.user, project: s.project, role: roleOf(s) })
  }
  const clear = () => {
    storage.set(undefined)
    set({ status: 'signedOut', accessToken: undefined, refreshToken: undefined, user: undefined, project: null, memberships: [], role: undefined })
  }
  const loadMemberships = async () => {
    try {
      const me = await authApi.me()
      set({ memberships: me.memberships })
    } catch {
      /* daftar project hanya untuk pilihan ganti project */
    }
  }

  return {
    status: 'restoring',
    project: null,
    memberships: [],

    login: async (email, password) => {
      apply(await authApi.login(email, password))
      await loadMemberships()
    },

    // Dipanggil sekali saat aplikasi dibuka. Refresh token berganti tiap dipakai, dan memakai token lama
    // dianggap pencurian oleh core-api → jangan pernah dijalankan dua kali bersamaan.
    restore: () => {
      restoring ??= (async () => {
        if (get().status === 'signedIn') return // sesi cetak sementara (printToken.ts)
        const token = storage.get()
        if (!token) return clear()
        try {
          apply(await authApi.refresh(token))
          await loadMemberships()
        } catch {
          clear()
        }
      })()
      return restoring
    },

    refresh: async () => {
      const token = get().refreshToken
      if (!token) return false
      try {
        apply(await authApi.refresh(token))
        return true
      } catch {
        return false
      }
    },

    switchProject: async (projectId) => {
      const token = get().refreshToken
      if (!token) return
      apply(await authApi.switchProject(token, projectId))
    },

    logout: async () => {
      const token = get().refreshToken
      clear()
      if (token) await authApi.logout(token).catch(() => undefined)
    },
  }
})

setAuthBridge({
  token: () => useAuthStore.getState().accessToken,
  refresh: () => useAuthStore.getState().refresh(),
  expired: () => {
    storage.set(undefined)
    useAuthStore.setState({ status: 'signedOut', accessToken: undefined, refreshToken: undefined })
  },
})
