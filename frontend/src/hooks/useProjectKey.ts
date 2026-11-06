import { useAuthStore } from '@/stores/auth.store'

/** Kunci cache yang selalu menyertakan project aktif → ganti project = data baru. */
export const useProjectKey = (...parts: unknown[]) => {
  const pid = useAuthStore((s) => s.project?.project_id)
  return { pid, key: [pid, ...parts] as const }
}
