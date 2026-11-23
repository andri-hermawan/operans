// Input harian core-api (ADR 0007). Pola sama untuk semua: daftar per tanggal (+ shift), tambah, ubah, hapus.
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type {
  EquipmentActivation,
  EquipmentBreakdown,
  FuelFilling,
  OperatorAssignment,
  RainSlipperyLog,
  Ritase,
  StockEmplacement,
  TimeSheet,
  WeighbridgeTicket,
  WorkHourNotification,
} from '@/types/daily.types'

const dailyApi = <T extends { id: number }>(path: string) => ({
  /** Satu tanggal operasional (dan satu shift bila shiftId diisi). */
  list: async (projectId: string, date?: string, shiftId?: string) =>
    (await core.get<ListResponse<T>>(path, { params: { project_id: projectId, date_from: date, date_to: date, shift_id: shiftId } })).data,
  create: (body: Record<string, unknown>) => core.post(path, body),
  update: (id: number, body: Record<string, unknown>) => core.patch(`${path}/${id}`, body),
  remove: (id: number) => core.del(`${path}/${id}`),
})

export const operatorApi = dailyApi<OperatorAssignment>('/api/v1/operator-assignments')
export const activationApi = dailyApi<EquipmentActivation>('/api/v1/equipment-activations')
const breakdown = dailyApi<EquipmentBreakdown>('/api/v1/equipment-breakdowns')
export const breakdownApi = {
  ...breakdown,
  /** Rentang tanggal operasional (+ shift dan kode asset bila diisi) — untuk Report → Breakdown History. */
  range: async (p: { projectId: string; from?: string; to?: string; shiftId?: string; equipmentCode?: string }) =>
    (await core.get<ListResponse<EquipmentBreakdown>>('/api/v1/equipment-breakdowns', {
      params: { project_id: p.projectId, date_from: p.from, date_to: p.to, shift_id: p.shiftId, equipment_code: p.equipmentCode },
    })).data,
}
export const ticketApi = dailyApi<WeighbridgeTicket>('/api/v1/weighbridge-tickets')
export const timeSheetApi = dailyApi<TimeSheet>('/api/v1/time-sheets')
const fuelFilling = dailyApi<FuelFilling>('/api/v1/fuel-fillings')
export const fuelFillingApi = {
  ...fuelFilling,
  /** Rentang tanggal operasional (+ shift dan kode asset bila diisi) — untuk Report → Fuel Filling. */
  range: async (p: { projectId: string; from?: string; to?: string; shiftId?: string; equipmentCode?: string }) =>
    (await core.get<ListResponse<FuelFilling>>('/api/v1/fuel-fillings', {
      params: { project_id: p.projectId, date_from: p.from, date_to: p.to, shift_id: p.shiftId, equipment_code: p.equipmentCode },
    })).data,
}
export const ritaseApi = dailyApi<Ritase>('/api/v1/ritases')
export const workHourApi = dailyApi<WorkHourNotification>('/api/v1/work-hour-notifications')
export const rainSlipperyApi = dailyApi<RainSlipperyLog>('/api/v1/rain-slippery-logs')

const stock = dailyApi<StockEmplacement>('/api/v1/stock-emplacements')
/** Stok cargo: satu tanggal operasional, tanpa shift. */
export const stockApi = {
  ...stock,
  list: async (projectId: string, from?: string, to?: string) =>
    (await core.get<ListResponse<StockEmplacement>>('/api/v1/stock-emplacements', { params: { project_id: projectId, date_from: from, date_to: to } })).data,
}
