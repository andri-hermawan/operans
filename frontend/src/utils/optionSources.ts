// Sumber pilihan select yang dimuat dari API (dipakai di FieldDef.options, lihat hooks/useOptions.ts).
import { listAlertTypes } from '@/services/api/alertType.api'
import { listCompanies } from '@/services/api/company.api'
import { listUsers } from '@/services/api/user.api'
import { getZones } from '@/services/api/zone.api'
import type { Option } from '@/types/form.types'

export const companyOptions = {
  key: 'company-options',
  load: async (): Promise<Option[]> => (await listCompanies()).map((c) => ({ value: c.id, label: `${c.company_code} — ${c.company_name}` })),
}

/** Penerima WhatsApp: hanya user aktif yang punya nomor HP. */
export const recipientOptions = {
  key: 'wa-recipients',
  load: async (pid: string): Promise<Option[]> =>
    (await listUsers(pid)).filter((u) => u.is_active && u.phone_number).map((u) => ({ value: u.id, label: `${u.name} (${u.phone_number})` })),
}

/** Nama area peta yang berlaku (untuk aturan alert per zona). */
export const zoneNameOptions = {
  key: 'zone-names',
  load: async (pid: string): Promise<Option[]> => {
    const fc = await getZones(pid)
    return [...new Set(fc.features.map((f) => f.properties.name))].sort().map((n) => ({ value: n, label: n }))
  },
}

export const alertTypeOptions = {
  key: 'alert-types',
  load: async (): Promise<Option[]> => (await listAlertTypes()).map((t) => ({ value: t.code, label: `${t.legacy_code} — ${t.name}` })),
}
