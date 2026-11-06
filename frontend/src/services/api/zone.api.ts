// Peta area project yang berlaku (GeoJSON, core-api).
import { core } from '@/services/http/client'
import type { ZoneCollection } from '@/types/zone.types'

export const getZones = (projectId: string) => core.get<ZoneCollection>('/api/v1/geo/zones', { params: { project_id: projectId } })
