// Kalibrasi sensor fuel per unit (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { FuelCalibration } from '@/types/fuelCalibration.types'

export const listFuelCalibrations = async (projectId: string) =>
  (await core.get<ListResponse<FuelCalibration>>('/api/v1/fuel-calibrations', { params: { project_id: projectId } })).data
export const saveFuelCalibration = (equipmentId: string, body: Record<string, unknown>) =>
  core.put(`/api/v1/equipment/${equipmentId}/fuel-calibration`, body)
export const deleteFuelCalibration = (equipmentId: string) => core.del(`/api/v1/equipment/${equipmentId}/fuel-calibration`)
