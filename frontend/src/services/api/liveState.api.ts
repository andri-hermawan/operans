// Kondisi terakhir semua unit (layanan IoT).
import { iot } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { LiveState } from '@/types/liveState.types'

export const listLiveStates = async () => (await iot.get<ListResponse<LiveState>>('/api/v1/equipment-live-states')).data
