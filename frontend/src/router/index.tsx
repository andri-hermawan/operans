import { Result } from 'antd'
import { createBrowserRouter, Navigate } from 'react-router-dom'

import type { PageRoute } from '@/types/route.types'
import RouteError from '@/components/feedback/RouteError'
import ComingSoon from '@/components/ui/ComingSoon'
import MainLayout from '@/layouts/MainLayout'
import { BASE_PATH } from '@/utils/basePath'
import { RequireAuth } from '@/router/guards/AuthGuard'
import { RequireAccess } from '@/router/guards/RoleGuard'
import LoginPage from '@/pages/auth/LoginPage'

import { HOME, ROUTES } from './routes'

const element = (r: PageRoute) => {
  const Page = r.page
  return <RequireAccess route={r}>{Page ? <Page /> : <ComingSoon title={r.title} phase={r.phase} />}</RequireAccess>
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  // halaman cetak tanpa menu (PDF jadwal laporan); login lewat sesi biasa atau token cetak
  { path: '/print/report-alert-summary', lazy: async () => ({ Component: (await import('@/pages/print/PrintReportAlertSummaryPage')).default }) },
  { path: '/print/report-executive-hauling', lazy: async () => ({ Component: (await import('@/pages/print/PrintReportExecutiveHaulingPage')).default }) },
  { path: '/print/report-executive-emplacement', lazy: async () => ({ Component: (await import('@/pages/print/PrintReportExecutiveEmplacementPage')).default }) },
  {
    path: '/',
    element: (
      <RequireAuth>
        <MainLayout />
      </RequireAuth>
    ),
    errorElement: <RouteError />,
    children: [
      { index: true, element: <Navigate to={HOME} replace /> },
      ...ROUTES.map((r) => ({ path: r.path.slice(1), element: element(r) })),
      { path: '*', element: <Result status="404" title="Halaman tidak ditemukan" /> },
    ],
  },
], { basename: BASE_PATH })
