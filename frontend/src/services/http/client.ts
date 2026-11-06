// HTTP client tunggal untuk semua backend. Setiap services/api/*.api.ts memakai `core` atau `iot` dari sini.
// - menambah header Authorization
// - bila 401: perpanjang sesi SEKALI (satu proses refresh untuk semua permintaan) lalu ulangi
// - galat selalu berbentuk ApiError dengan pesan dari server ({"error": "..."})
import type { AuthBridge, Query, RequestOptions } from '@/types/api.types'

import { ApiError } from './ApiError'

let bridge: AuthBridge = { token: () => undefined, refresh: async () => false, expired: () => {} }
export const setAuthBridge = (b: AuthBridge) => {
  bridge = b
}

let refreshing: Promise<boolean> | null = null
const refreshOnce = () => {
  refreshing ??= bridge.refresh().finally(() => {
    refreshing = null
  })
  return refreshing
}

const buildUrl = (base: string, path: string, params?: Query) => {
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params ?? {})) {
    if (v === undefined || v === null || v === '') continue
    if (Array.isArray(v)) v.forEach((x) => qs.append(k, String(x)))
    else qs.append(k, String(v))
  }
  const q = qs.toString()
  return `${base}${path}${q ? `?${q}` : ''}`
}

async function request<T>(base: string, method: string, path: string, opt: RequestOptions = {}, retried = false): Promise<T> {
  const headers: Record<string, string> = {}
  const token = opt.anonymous ? undefined : bridge.token()
  if (token) headers.Authorization = `Bearer ${token}`
  let body: BodyInit | undefined
  if (opt.body instanceof FormData) body = opt.body
  else if (opt.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opt.body)
  }

  let res: Response
  try {
    res = await fetch(buildUrl(base, path, opt.params), { method, headers, body, signal: opt.signal })
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e
    throw new ApiError(0, 'Tidak dapat terhubung ke server. Periksa jaringan atau layanan backend.')
  }

  if (res.status === 401 && !opt.anonymous && !retried) {
    if (await refreshOnce()) return request<T>(base, method, path, opt, true)
    bridge.expired()
    throw new ApiError(401, 'Sesi berakhir, silakan login kembali.')
  }
  if (!res.ok) {
    let message = `Permintaan gagal (HTTP ${res.status})`
    let data: unknown
    try {
      const json = await res.json()
      if (json?.error) message = String(json.error)
      data = json?.data
    } catch {
      /* bukan JSON */
    }
    throw new ApiError(res.status, message, data)
  }
  if (res.status === 204) return undefined as T
  if (opt.blob) return (await res.blob()) as T
  // beberapa endpoint (simpan) membalas 200/201 tanpa isi
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

const client = (base: string) => ({
  get: <T>(path: string, opt?: RequestOptions) => request<T>(base, 'GET', path, opt),
  post: <T>(path: string, body?: unknown, opt?: RequestOptions) => request<T>(base, 'POST', path, { ...opt, body }),
  put: <T>(path: string, body?: unknown, opt?: RequestOptions) => request<T>(base, 'PUT', path, { ...opt, body }),
  patch: <T>(path: string, body?: unknown, opt?: RequestOptions) => request<T>(base, 'PATCH', path, { ...opt, body }),
  del: <T = void>(path: string, opt?: RequestOptions) => request<T>(base, 'DELETE', path, opt),
})

/** core-api (auth, master, input harian, peta, aturan). */
export const core = client('/core')
/** layanan IoT (live state, telemetry, alert, kejadian). */
export const iot = client('/iot')
/** layanan AI Brain (dokumen SOP, pencarian). Jawabannya apa adanya (bukan dibungkus { data }). */
export const ai = client('/ai')
