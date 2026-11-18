import { useQuery } from '@tanstack/react-query'

import { listOrthoVersions } from '@/services/api/ortho.api'

/** Versi citra ortho yang tersedia (terbaru di akhir daftar). Kosong bila belum ada. */
export function useOrthoVersions() {
  return useQuery({
    queryKey: ['ortho-versions'],
    staleTime: 10 * 60_000,
    retry: false,
    queryFn: listOrthoVersions,
  })
}
