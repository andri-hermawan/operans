// Bentuk umum jawaban backend SIMHA.

/** Daftar standar: { data: [...], count? }. */
export interface ListResponse<T> {
  data: T[]
  count?: number
}

/** Daftar dari layanan IoT yang bisa terpotong (batas baris). */
export type TruncatedList<T> = ListResponse<T> & { truncated?: boolean }

/** Parameter query string: nilai kosong dibuang, array diulang (?a=1&a=2). */
export type Query = Record<string, string | number | boolean | null | undefined | (string | number)[]>

export interface RequestOptions {
  params?: Query
  body?: unknown // objek → JSON, FormData → multipart
  signal?: AbortSignal
  /** Tanpa token & tanpa refresh otomatis (dipakai oleh /auth/*). */
  anonymous?: boolean
  /** Ambil sebagai Blob (unduh file). */
  blob?: boolean
}

/** Dipasang oleh auth store agar http tidak bergantung langsung pada store (hindari import melingkar). */
export interface AuthBridge {
  token: () => string | undefined
  refresh: () => Promise<boolean>
  expired: () => void
}
