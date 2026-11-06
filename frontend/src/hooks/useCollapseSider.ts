import { useEffect } from 'react'

import { useUiStore } from '@/stores/ui.store'

/** Halaman peta penuh (Live Tracking, Geofencing): saat dibuka, menu samping ditutup agar peta lebih lebar (bisa dibuka lagi lewat tombol menu). */
export function useCollapseSider() {
  const setCollapsed = useUiStore((s) => s.setSiderCollapsed)
  useEffect(() => setCollapsed(true), [setCollapsed])
}
