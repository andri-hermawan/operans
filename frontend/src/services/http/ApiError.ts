// Galat API: selalu membawa status HTTP dan pesan dari server ({"error": "..."}).

export class ApiError extends Error {
  readonly status: number
  /** Isi "data" dari jawaban galat (mis. daftar baris salah pada impor 422). */
  readonly data?: unknown
  constructor(status: number, message: string, data?: unknown) {
    super(message)
    this.status = status
    this.data = data
  }
}

/** Pesan galat siap tampil dari error apa pun. */
export const errorMessage = (e: unknown): string =>
  e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Terjadi kesalahan tak terduga'
