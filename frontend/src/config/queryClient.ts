import { QueryClient } from '@tanstack/react-query'

import { ApiError } from '@/services/http/ApiError'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,
      refetchOnWindowFocus: false,
      // galat 4xx (akses ditolak, data salah) tidak perlu diulang
      retry: (count, e) => !(e instanceof ApiError && e.status >= 400 && e.status < 500) && count < 1,
    },
  },
})
