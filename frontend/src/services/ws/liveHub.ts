// Satu koneksi WebSocket peta live (/iot/ws/live) untuk SELURUH aplikasi, dipakai bersama oleh header (lonceng alert)
// dan halaman-halaman peta. Koneksi dibuka saat ada pemakai pertama dan ditutup saat pemakai terakhir pergi.
// - login dengan pesan pertama {"type":"auth","token":…}
// - galat 4401 (token habis) → sesi diperpanjang lalu sambung ulang; putus lain → sambung ulang bertahap (1, 2, 4 … 32 dtk)
// Modul ini tidak tahu soal React Query/store: pemakai memberi `auth` dan penerima pesan.
import type { LiveStatus } from '@/types/liveState.types'

export interface LiveMessage {
  type: string
  data: unknown
}
export type LiveListener = (msg: LiveMessage) => void
export interface LiveAuth {
  getToken: () => string | null | undefined
  /** Perpanjang sesi (dipanggil saat token habis). */
  refresh: () => Promise<unknown>
}

let ws: WebSocket | undefined
let timer: ReturnType<typeof setTimeout> | undefined
let retry = 0
let refs = 0
let auth: LiveAuth | undefined
let status: LiveStatus = 'connecting'
const listeners = new Set<LiveListener>()
const statusSubs = new Set<() => void>()

const setStatus = (s: LiveStatus) => {
  if (status === s) return
  status = s
  statusSubs.forEach((f) => f())
}

/** Untuk useSyncExternalStore. */
export const getLiveStatus = (): LiveStatus => status
export const subscribeLiveStatus = (f: () => void) => {
  statusSubs.add(f)
  return () => {
    statusSubs.delete(f)
  }
}

function connect() {
  setStatus('connecting')
  const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/iot/ws/live`
  const socket = new WebSocket(url)
  ws = socket
  socket.onopen = () => socket.send(JSON.stringify({ type: 'auth', token: auth?.getToken() }))
  socket.onmessage = (ev) => {
    const msg = JSON.parse(ev.data) as LiveMessage
    if (msg.type === 'ready') {
      retry = 0
      setStatus('live')
    }
    listeners.forEach((l) => l(msg))
  }
  socket.onclose = async (ev) => {
    if (ws !== socket || refs === 0) return // sudah diganti / sengaja ditutup
    setStatus('offline')
    if (ev.code === 4401) await auth?.refresh() // token habis → perpanjang dulu
    retry = Math.min(retry + 1, 6)
    timer = setTimeout(() => refs > 0 && connect(), 1000 * 2 ** (retry - 1))
  }
}

/** Ikut memakai koneksi bersama. Mengembalikan fungsi untuk berhenti. */
export function acquireLive(a: LiveAuth, listener: LiveListener): () => void {
  auth = a
  listeners.add(listener)
  refs++
  if (refs === 1) {
    retry = 0
    connect()
  }
  return () => {
    listeners.delete(listener)
    refs--
    if (refs === 0) {
      clearTimeout(timer)
      const socket = ws
      ws = undefined
      socket?.close()
    }
  }
}
