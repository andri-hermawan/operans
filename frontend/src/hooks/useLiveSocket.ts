// WebSocket peta live (layanan IoT /ws/live, ADR 0005/A4). Satu koneksi bersama untuk seluruh aplikasi
// (services/ws/liveHub.ts); hook ini hanya ikut mendengarkan:
// - "equipment_state" → kondisi unit di cache diperbarui (peta bergerak tanpa memuat ulang)
// - "alert" / "geofence_event" → daftar terkait dimuat ulang + penerima khusus halaman (handlers) dipanggil
import { useEffect, useRef, useSyncExternalStore } from 'react'
import { useQueryClient } from '@tanstack/react-query'

import { acquireLive, getLiveStatus, subscribeLiveStatus } from '@/services/ws/liveHub'
import { useAuthStore } from '@/stores/auth.store'
import type { AlertRow } from '@/types/alert.types'
import type { GeofenceEvent } from '@/types/geofence.types'
import type { LiveHandlers, LiveState, LiveStatus } from '@/types/liveState.types'

import { liveKey } from './useLiveStates'

export function useLiveSocket(handlers: LiveHandlers = {}): LiveStatus {
  const qc = useQueryClient()
  const pid = useAuthStore((s) => s.project?.project_id)
  const h = useRef(handlers)
  useEffect(() => {
    h.current = handlers
  })

  useEffect(() => {
    if (!pid) return
    const auth = {
      getToken: () => useAuthStore.getState().accessToken,
      refresh: () => useAuthStore.getState().refresh(),
    }
    return acquireLive(auth, (msg) => {
      switch (msg.type) {
        case 'ready':
          qc.invalidateQueries({ queryKey: liveKey(pid) }) // isi ulang setelah (sambung ulang)
          break
        case 'equipment_state': {
          const st = msg.data as LiveState
          qc.setQueryData<LiveState[]>(liveKey(pid), (old) => {
            if (!old) return old
            const i = old.findIndex((x) => x.equipment_id === st.equipment_id)
            if (i < 0) return [...old, st]
            const next = old.slice()
            next[i] = st
            return next
          })
          break
        }
        case 'alert':
          h.current.onAlert?.(msg.data as AlertRow)
          qc.invalidateQueries({ queryKey: [pid, 'alerts'] })
          break
        case 'geofence_event':
          h.current.onGeofence?.(msg.data as GeofenceEvent)
          qc.invalidateQueries({ queryKey: [pid, 'geofence-events'] })
          break
      }
    })
  }, [pid, qc])

  return useSyncExternalStore(subscribeLiveStatus, getLiveStatus)
}
