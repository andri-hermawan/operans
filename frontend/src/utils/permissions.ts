// Hak akses per role — satu tabel untuk menu, route, dan tombol (sesuai hak backend core-api ADR 0006/0007).
import type { Action, Resource, Role } from '@/types/auth.types'

/** Kelompok sumber data → role yang boleh melakukan aksi. */
const RULES: Record<Resource, Partial<Record<Action, Role[]>>> = {
  monitoring: { view: ['superadmin', 'admin', 'dispatcher', 'viewer'], export: ['superadmin', 'admin', 'dispatcher', 'viewer'] },
  report: { view: ['superadmin', 'admin', 'dispatcher', 'viewer'], export: ['superadmin', 'admin', 'dispatcher', 'viewer'] },
  // input harian (operator, aktivasi, breakdown, weighbridge): admin & dispatcher boleh menulis
  daily: {
    view: ['superadmin', 'admin', 'dispatcher', 'viewer'],
    create: ['superadmin', 'admin', 'dispatcher'],
    update: ['superadmin', 'admin', 'dispatcher'],
    delete: ['superadmin', 'admin', 'dispatcher'],
    import: ['superadmin', 'admin', 'dispatcher'],
    export: ['superadmin', 'admin', 'dispatcher', 'viewer'],
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
  // dokumen SOP (AI Brain): semua role membaca; admin (project aktif) & superadmin mengelola. Hak per dokumen diatur di backend.
  documents: {
    view: ['superadmin', 'admin', 'dispatcher', 'viewer'],
    create: ['superadmin', 'admin'],
    update: ['superadmin', 'admin'],
    delete: ['superadmin', 'admin'],
    export: ['superadmin'], // cetak PDF asli: hanya superadmin (admin project pun tidak)
  },
  // company & project hanya superadmin yang mengubah
  organization: { view: ['superadmin', 'admin'], create: ['superadmin'], update: ['superadmin'], delete: ['superadmin'] },
}

export const can = (role: Role | undefined, resource: Resource, action: Action = 'view'): boolean =>
  !!role && !!RULES[resource]?.[action]?.includes(role)
