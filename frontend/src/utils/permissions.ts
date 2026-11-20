// Hak akses per role — satu tabel untuk menu, route, dan tombol (sesuai hak backend core-api ADR 0006/0007).
import type { Action, Resource, Role } from '@/types/auth.types'

/** Kelompok sumber data → role yang boleh melakukan aksi. */
const RULES: Record<Resource, Partial<Record<Action, Role[]>>> = {
  monitoring: { view: ['superadmin', 'admin', 'dispatcher', 'viewer'], export: ['superadmin', 'admin', 'dispatcher', 'viewer'] },
  // report, Record/Upload, dan Dokumen SOP tidak ditampilkan untuk viewer (2026-11-19); viewer hanya monitoring
  report: { view: ['superadmin', 'admin', 'dispatcher'], export: ['superadmin', 'admin', 'dispatcher'] },
  // input harian (operator, aktivasi, breakdown, weighbridge): admin & dispatcher boleh menulis
  daily: {
    view: ['superadmin', 'admin', 'dispatcher'],
    create: ['superadmin', 'admin', 'dispatcher'],
    update: ['superadmin', 'admin', 'dispatcher'],
    delete: ['superadmin', 'admin', 'dispatcher'],
    import: ['superadmin', 'admin', 'dispatcher'],
    export: ['superadmin', 'admin', 'dispatcher'],
  },
  // data master & pengaturan: admin (project aktif) & superadmin
  master: {
    view: ['superadmin', 'admin'],
    create: ['superadmin', 'admin'],
    update: ['superadmin', 'admin'],
    delete: ['superadmin', 'admin'],
    import: ['superadmin', 'admin'],
    export: ['superadmin', 'admin'],
  },
  // dokumen SOP (AI Brain): superadmin, admin & dispatcher membaca (viewer tidak); admin (project aktif) & superadmin mengelola. Hak per dokumen diatur di backend.
  documents: {
    view: ['superadmin', 'admin', 'dispatcher'],
    create: ['superadmin', 'admin'],
    update: ['superadmin', 'admin'],
    delete: ['superadmin', 'admin'],
    export: ['superadmin'], // cetak PDF asli: hanya superadmin (admin project pun tidak)
  },
  // pengaturan hak akses halaman & laporan: hanya superadmin
  access: { view: ['superadmin'] },
  // company & project hanya superadmin yang mengubah
  organization: { view: ['superadmin', 'admin'], create: ['superadmin'], update: ['superadmin'], delete: ['superadmin'] },
}

export const can = (role: Role | undefined, resource: Resource, action: Action = 'view'): boolean =>
  !!role && !!RULES[resource]?.[action]?.includes(role)

// ---------------------------------------------------------------------
// Hak akses halaman yang diatur superadmin (per project, per role / per user — ADR 0018)
// ---------------------------------------------------------------------

/** Perubahan dari aturan bawaan untuk user ini di project aktif: kunci halaman → boleh (true) / diblokir (false). */
export type AccessOverrides = Record<string, boolean>

/** Kunci halaman = alamat tanpa "/" di depan, mis. "/report/report-ritase-daily" → "report/report-ritase-daily". */
export const pageKey = (path: string): string => path.replace(/^\//, '')

/**
 * Halaman yang boleh DIBUKAKAN lewat pengaturan. Data master, organisasi, dan pengaturan akses dilayani API khusus admin,
 * jadi tidak bisa dibukakan untuk role lain (hanya bisa dibatasi).
 */
export const isGrantable = (resource: Resource): boolean => resource !== 'master' && resource !== 'organization' && resource !== 'access'

/** Bolehkah role ini membuka halaman? Aturan bawaan (tabel RULES), lalu diubah oleh pengaturan superadmin bila ada. */
export function canOpenPage(role: Role | undefined, resource: Resource, key: string | false, overrides?: AccessOverrides): boolean {
  if (!role) return false
  if (role === 'superadmin' || key === false) return can(role, resource)
  const set = overrides?.[key]
  if (set === false) return false
  if (set === true) return isGrantable(resource) || can(role, resource)
  return can(role, resource)
}
