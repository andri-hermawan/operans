import { beforeEach, describe, expect, it } from 'vitest'

import type { AlertRow } from '@/types/alert.types'

import { useAlertStore } from './alert.store'

const alert = (id: number, extra: Partial<AlertRow> = {}): AlertRow =>
  ({ id, equipment_id: 'e1', alert_type: 'overspeed', started_at: '2026-10-28T01:00:00Z', ended_at: null, ...extra }) as AlertRow

const A = 'userA|p1'
const B = 'userB|p1'
const ids = (scope: string) => {
  const s = useAlertStore.getState()
  return s.scope === scope ? s.inbox.map((a) => a.id) : []
}

describe('alert.store (notifikasi lonceng)', () => {
  beforeEach(() => useAlertStore.setState({ seen: {}, scope: undefined, inbox: [] }))

  it('alert dari WebSocket langsung masuk, terbaru di atas, id sama diganti (alert diperpanjang)', () => {
    const s = useAlertStore.getState()
    s.receive(A, alert(1))
    s.receive(A, alert(2))
    s.receive(A, alert(1, { ended_at: '2026-10-28T01:05:00Z' }))
    expect(ids(A)).toEqual([2, 1])
    expect(useAlertStore.getState().inbox.find((a) => a.id === 1)?.ended_at).toBe('2026-10-28T01:05:00Z')
  })

  it('X mengosongkan daftar; yang tampil berikutnya hanya alert baru', () => {
    const s = useAlertStore.getState()
    s.receiveSnapshot(A, [alert(1), alert(2), alert(3)])
    s.clear(A)
    expect(ids(A)).toEqual([])
    s.receiveSnapshot(A, [alert(1), alert(2), alert(3)]) // muat ulang cadangan: yang lama tidak kembali
    s.receive(A, alert(3)) // pesan lama yang datang lagi
    expect(ids(A)).toEqual([])
    s.receive(A, alert(4))
    expect(ids(A)).toEqual([4])
  })

  it('per user: X milik user A tidak menghapus daftar user B', () => {
    const s = useAlertStore.getState()
    s.receiveSnapshot(A, [alert(1), alert(2)])
    s.clear(A) // user A klik X
    expect(useAlertStore.getState().seen[A]).toBe(2)
    s.receiveSnapshot(B, [alert(1), alert(2)]) // user B membuka: alert yang sama masih ada
    expect(ids(B)).toEqual([2, 1])
    s.clear(B) // sampai user B klik X juga
    expect(ids(B)).toEqual([])
    expect(useAlertStore.getState().seen[B]).toBe(2)
  })

  it('nomor alert mulai dari 1 lagi (data dikosongkan): penanda lama tidak menyembunyikan alert baru', () => {
    useAlertStore.setState({ seen: { [A]: 500 } })
    useAlertStore.getState().receiveSnapshot(A, [alert(1), alert(2)])
    expect(ids(A)).toEqual([2, 1])
  })
})
