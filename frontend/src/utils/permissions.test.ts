import { describe, expect, it } from 'vitest'

import { canOpenPage, isGrantable, pageKey } from './permissions'

describe('pageKey', () => {
  it('membuang "/" di depan', () => {
    expect(pageKey('/report/report-ritase-daily')).toBe('report/report-ritase-daily')
    expect(pageKey('/tracking')).toBe('tracking')
  })
})

describe('canOpenPage', () => {
  it('tanpa pengaturan = aturan bawaan', () => {
    expect(canOpenPage('viewer', 'monitoring', 'tracking')).toBe(true)
    expect(canOpenPage('viewer', 'report', 'report')).toBe(false)
    expect(canOpenPage('dispatcher', 'report', 'report')).toBe(true)
    expect(canOpenPage(undefined, 'monitoring', 'tracking')).toBe(false)
  })

  it('pengaturan memblokir halaman yang bawaannya boleh', () => {
    expect(canOpenPage('dispatcher', 'report', 'report/report-ritase-daily', { 'report/report-ritase-daily': false })).toBe(false)
    expect(canOpenPage('dispatcher', 'report', 'report/report-pa-daily', { 'report/report-ritase-daily': false })).toBe(true)
  })

  it('pengaturan membukakan halaman yang bawaannya tertutup (Report untuk viewer)', () => {
    expect(canOpenPage('viewer', 'report', 'report/report-breakdown-daily', { 'report/report-breakdown-daily': true })).toBe(true)
  })

  it('data master tidak bisa dibukakan untuk viewer/dispatcher, tetapi admin bisa dibatasi', () => {
    expect(isGrantable('master')).toBe(false)
    expect(canOpenPage('viewer', 'master', 'master/user', { 'master/user': true })).toBe(false)
    expect(canOpenPage('dispatcher', 'master', 'master/user', { 'master/user': true })).toBe(false)
    expect(canOpenPage('admin', 'master', 'master/alert-rule', { 'master/alert-rule': false })).toBe(false)
    expect(canOpenPage('admin', 'master', 'master/user', {})).toBe(true)
  })

  it('superadmin selalu penuh dan halaman tanpa kunci tidak bisa dibatasi', () => {
    expect(canOpenPage('superadmin', 'report', 'report/x', { 'report/x': false })).toBe(true)
    expect(canOpenPage('viewer', 'monitoring', false, { profile: false })).toBe(true)
  })

  it('Hak Akses hanya untuk superadmin', () => {
    expect(canOpenPage('superadmin', 'access', 'master/access')).toBe(true)
    expect(canOpenPage('admin', 'access', 'master/access')).toBe(false)
    expect(canOpenPage('admin', 'access', 'master/access', { 'master/access': true })).toBe(false)
  })
})
