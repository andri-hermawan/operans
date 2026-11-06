// Keadaan tampilan kecil yang dipakai bersama (menu samping dilipat).
import { create } from 'zustand'

interface UiState {
  siderCollapsed: boolean
  setSiderCollapsed: (v: boolean) => void
  toggleSider: () => void
}

export const useUiStore = create<UiState>((set) => ({
  siderCollapsed: typeof window !== 'undefined' && window.innerWidth < 768,
  setSiderCollapsed: (v) => set({ siderCollapsed: v }),
  toggleSider: () => set((s) => ({ siderCollapsed: !s.siderCollapsed })),
}))
