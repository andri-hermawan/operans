// Alat GPS (layanan IoT).
import { iot } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Device } from '@/types/device.types'

export const listDevices = async () => (await iot.get<ListResponse<Device>>('/api/v1/devices')).data
export const createDevice = (body: Record<string, unknown>) => iot.post('/api/v1/devices', body)
export const updateDevice = (id: string, body: Record<string, unknown>) => iot.patch(`/api/v1/devices/${id}`, body)
/** API key baru untuk alat HTTP/simulator; hanya ditampilkan sekali. */
export const rotateDeviceApiKey = async (id: string) => (await iot.post<{ api_key: string }>(`/api/v1/devices/${id}/api-key`)).api_key
