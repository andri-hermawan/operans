import { useAuthStore } from '@/stores/auth.store'
import type { Action, Resource } from '@/types/auth.types'
import { can } from '@/utils/permissions'

/** Hook untuk komponen: const allowed = useCan('daily', 'create'). */
export const useCan = (resource: Resource, action: Action = 'view'): boolean => can(useAuthStore((s) => s.role), resource, action)
