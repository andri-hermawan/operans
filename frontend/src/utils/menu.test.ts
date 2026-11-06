import { describe, expect, it } from 'vitest'

import type { MenuEntry, PageRoute } from '@/types/route.types'

import { groupKeysTo, isGroup, pagesOf, pruneMenu } from './menu'

const page = (path: string, resource: PageRoute['resource'] = 'daily'): PageRoute => ({ path, title: path, resource })

// Bentuk sama dengan menu Record / Upload: dua kelompok di dalam satu kelompok.
const MENU: MenuEntry[] = [
  page('/dashboard', 'monitoring'),
  {
    key: 'upload',
    title: 'Record / Upload',
    icon: null,
    children: [
      { key: 'transaction', title: 'Transaction', icon: null, children: [page('/upload/weighbridge'), page('/upload/stock-emplacement')] },
      { key: 'plan', title: 'Plan', icon: null, children: [page('/upload/plan-yearly'), page('/upload/plan-monthly')] },
    ],
  },
  { key: 'master', title: 'System Data', icon: null, children: [page('/master/user', 'master')] },
]

describe('menu bersarang', () => {
  it('isGroup membedakan kelompok dan halaman', () => {
    expect(isGroup(MENU[0])).toBe(false)
    expect(isGroup(MENU[1])).toBe(true)
  })

  it('pagesOf meratakan semua tingkat', () => {
    const paths = MENU.flatMap(pagesOf).map((p) => p.path)
    expect(paths).toEqual(['/dashboard', '/upload/weighbridge', '/upload/stock-emplacement', '/upload/plan-yearly', '/upload/plan-monthly', '/master/user'])
  })

  it('groupKeysTo membuka kelompok luar dan dalam', () => {
    expect(groupKeysTo(MENU, '/upload/plan-yearly')).toEqual(['upload', 'plan'])
    expect(groupKeysTo(MENU, '/upload/weighbridge')).toEqual(['upload', 'transaction'])
    expect(groupKeysTo(MENU, '/master/user')).toEqual(['master'])
    expect(groupKeysTo(MENU, '/dashboard')).toEqual([])
    expect(groupKeysTo(MENU, '/tidak-ada')).toEqual([])
  })

  it('pruneMenu menyembunyikan halaman terlarang dan kelompok yang jadi kosong, di tingkat mana pun', () => {
    const onlyMonitoring = pruneMenu(MENU, (p) => p.resource === 'monitoring')
    expect(onlyMonitoring.map((e) => ('path' in e ? e.path : e.key))).toEqual(['/dashboard']) // Record / Upload & System Data hilang seluruhnya

    // kelompok dalam kosong, kelompok dalam lain tetap → kelompok luar tetap tampil hanya dengan yang tersisa
    const noPlan = pruneMenu(MENU, (p) => !p.path.includes('/plan-'))
    const upload = noPlan.find((e) => isGroup(e) && e.key === 'upload')
    expect(upload && isGroup(upload) && upload.children.map((c) => ('key' in c ? c.key : c.path))).toEqual(['transaction'])
  })

  it('pruneMenu tidak mengubah menu asli', () => {
    pruneMenu(MENU, () => false)
    expect(MENU.flatMap(pagesOf)).toHaveLength(6)
  })
})
