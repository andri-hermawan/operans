// Notifikasi alert di ikon lonceng header.
// - `inbox`  = alert yang belum dibaca (di memori): diisi dari WebSocket secara langsung, dan dari daftar hari ini
//              saat halaman dibuka. Menutup laci notifikasi (tombol X) MENGOSONGKAN inbox.
// - `seen`   = id alert terbesar yang sudah dikosongkan, per "user|project" (disimpan di browser). Alert dengan id
//              di bawah/sama dengan ini tidak muncul lagi, jadi setelah X yang tampil hanya alert baru.
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { AlertRow } from '@/types/alert.types'

const MAX_INBOX = 200

interface AlertState {
  seen: Record<string, number>
  scope?: string
  inbox: AlertRow[]
  /** Alert baru dari WebSocket. */
  receive: (scope: string, alert: AlertRow) => void
  /** Daftar alert hari ini saat halaman dibuka / dimuat ulang (menggantikan isi yang sudah ada). */
  receiveSnapshot: (scope: string, alerts: AlertRow[]) => void
  /** X: kosongkan daftar dan catat id terbesar sebagai sudah dibaca. */
  clear: (scope: string) => void
}

const merge = (inbox: AlertRow[], rows: AlertRow[], seen: number): AlertRow[] => {
  const byId = new Map(inbox.map((a) => [a.id, a]))
  for (const a of rows) if (a.id > seen) byId.set(a.id, a) // alert yang sama (mis. diperpanjang) diganti
  return [...byId.values()].sort((a, b) => b.id - a.id).slice(0, MAX_INBOX)
}

export const useAlertStore = create<AlertState>()(
  persist(
    (set) => ({
      seen: {},
      inbox: [],
      receive: (scope, alert) =>
        set((s) => {
          const inbox = s.scope === scope ? s.inbox : []
          return { scope, inbox: merge(inbox, [alert], s.seen[scope] ?? 0) }
        }),
      receiveSnapshot: (scope, alerts) =>
        set((s) => {
          let seen = s.seen[scope] ?? 0
          // nomor alert bisa mulai dari 1 lagi (data dikosongkan): penanda yang lebih besar dari semua nomor tidak berlaku
          const maxId = alerts.reduce((m, a) => Math.max(m, a.id), 0)
          if (alerts.length > 0 && seen > maxId) seen = 0
          const inbox = s.scope === scope ? s.inbox : []
          return { scope, seen: { ...s.seen, [scope]: seen }, inbox: merge(inbox, alerts, seen) }
        }),
      clear: (scope) =>
        set((s) => {
          const maxId = s.inbox.reduce((m, a) => Math.max(m, a.id), s.seen[scope] ?? 0)
          return { scope, inbox: [], seen: { ...s.seen, [scope]: maxId } }
        }),
    }),
    { name: 'simha.alert-seen', partialize: (s) => ({ seen: s.seen }) },
  ),
)
