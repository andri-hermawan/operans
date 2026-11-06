// Halaman cetak dibuka browser tanpa layar milik core-api (jadwal laporan) dengan "#token=…".
// Token dipasang sebagai sesi sementara SEBELUM aplikasi digambar, lalu dihapus dari alamat
// (bagian "#" tidak pernah dikirim ke server, jadi tidak tercatat di log).
import { useAuthStore } from '@/stores/auth.store'
import type { ProjectRole } from '@/types/auth.types'
import { appUrl } from '@/utils/basePath'

export function adoptPrintToken(): void {
  if (!window.location.pathname.startsWith(appUrl('/print/'))) return
  const token = new URLSearchParams(window.location.hash.slice(1)).get('token')
  if (!token) return
  window.history.replaceState(null, '', window.location.pathname + window.location.search)
  try {
    const claims = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'))) as {
      sub: string
      name?: string
      pid?: string
      prole?: ProjectRole
      sa?: boolean
    }
    useAuthStore.setState({
      status: 'signedIn',
      accessToken: token,
      user: { id: claims.sub, email: '', name: claims.name ?? 'Report', is_superadmin: !!claims.sa },
      project: claims.pid ? { project_id: claims.pid, project_code: '', project_name: '', role: claims.prole ?? 'viewer' } : null,
      role: claims.sa ? 'superadmin' : claims.prole,
    })
  } catch {
    window.__REPORT_ERROR__ = 'token cetak tidak valid'
  }
}

declare global {
  interface Window {
    __REPORT_READY__?: boolean
    __REPORT_ERROR__?: string
  }
}
