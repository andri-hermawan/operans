import { useMutation } from '@tanstack/react-query'

import { authApi } from '@/services/api/auth.api'
import type { PasswordForm } from '@/types/auth.types'

/** Ganti password sendiri. Halaman menentukan apa yang terjadi saat berhasil/gagal. */
export const useChangePassword = (opt: { onSuccess?: () => void | Promise<void>; onError?: (e: unknown) => void }) =>
  useMutation({
    mutationFn: (v: PasswordForm) => authApi.changePassword(v.current_password, v.new_password),
    onSuccess: opt.onSuccess,
    onError: opt.onError,
  })
