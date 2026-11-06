import { useCallback, useEffect, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'

import { listAlerts } from '@/services/api/alert.api'
import { useAlertStore } from '@/stores/alert.store'
import { useAuthStore } from '@/stores/auth.store'
import type { AlertRow } from '@/types/alert.types'

import { useCodeOf } from './useCodeOf'
import { useLiveSocket } from './useLiveSocket'
import { useOperationalDefaults } from './useOperationalDefaults'
import { useProjectKey } from './useProjectKey'

const SAFETY_REFRESH_MS = 60_000
const NONE: AlertRow[] = [] // rujukan tetap (selector tidak boleh membuat larik baru tiap pemanggilan)

/**
 * Data ikon lonceng di header.
 * - Alert baru masuk LANGSUNG dari WebSocket (koneksi bersama, aktif di semua halaman).
 * - Saat halaman dibuka, alert hari operasional ini yang belum dikosongkan ikut dimuat; dimuat ulang tiap 60 detik
 *   sebagai cadangan bila ada yang terlewat (mis. WebSocket sempat putus).
 * - `clear` (tombol X) mengosongkan daftar; yang tampil berikutnya hanya alert baru.
 */
export function useAlertBell() {
  const codeOf = useCodeOf()
  const now = useOperationalDefaults(true) // alert "hari ini" ikut berganti saat tanggal operasional berganti
  const userId = useAuthStore((s) => s.user?.id)
  const { pid, key } = useProjectKey('alerts', 'bell', now.date)
  const scope = `${userId}|${pid}`
  const { receive, receiveSnapshot, clear } = useAlertStore.getState()
  const inbox = useAlertStore((s) => (s.scope === scope ? s.inbox : NONE))

  useLiveSocket({ onAlert: (a) => receive(scope, a) })

  const snapshot = useQuery({
    queryKey: key,
    enabled: !!pid && now.ready,
    refetchInterval: SAFETY_REFRESH_MS,
    queryFn: () => listAlerts({ date: now.date }),
  })
  useEffect(() => {
    if (snapshot.data) receiveSnapshot(scope, snapshot.data.data)
  }, [snapshot.data, scope, receiveSnapshot])

  const alerts = useMemo(() => inbox.map((a) => ({ ...a, code: codeOf(a.equipment_id) })), [inbox, codeOf])
  const clearAll = useCallback(() => clear(scope), [clear, scope])

  return { alerts, unread: alerts.length, date: now.date, loading: snapshot.isLoading, clearAll }
}
