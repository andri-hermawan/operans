// SATU daftar halaman. Menu samping, judul tab, breadcrumb, dan hak akses semuanya dibaca dari sini.
// Menambah halaman = menambah satu baris di sini (urutan = urutan menu, sama dengan FMS lama).
import { lazy, type ComponentType } from 'react'
import {
  AlertTriangle,
  ArrowLeftRight,
  BarChart,
  BellRing,
  BookOpen,
  Building,
  Calendar,
  CalendarClock,
  CalendarDays,
  CalendarRange,
  ChartNoAxesCombined,
  CircleGauge,
  ClipboardClock,
  CloudRain,
  Clock,
  FileText,
  Fuel,
  Landmark,
  Hourglass,
  Layers,
  LayoutDashboard,
  Locate,
  Map as MapIcon,
  MapPin,
  OctagonAlert,
  PenTool,
  Repeat,
  Power,
  RadioReceiver,
  Boxes,
  Scale,
  Sliders,
  SquareUser,
  Truck,
  Upload,
  UserPlus,
} from 'lucide-react'

import type { Resource } from '@/types/auth.types'
import type { MenuEntry, Page, PageRoute } from '@/types/route.types'
import { pagesOf } from '@/utils/menu'


const ic = (C: ComponentType<{ size?: number }>) => <C size={18} />

// Halaman dimuat saat dibuka saja (kode tiap halaman jadi file terpisah; setiap halaman = satu file dengan default export).
const page = (load: () => Promise<{ default: ComponentType }>) => ({ page: lazy(load) })

export const MENU: MenuEntry[] = [
  { path: '/dashboard', title: 'Dashboard', icon: ic(LayoutDashboard), resource: 'monitoring', ...page(() => import('@/pages/dashboard/DashboardPage')) },
  { path: '/tracking', title: 'Live Tracking', icon: ic(MapIcon), resource: 'monitoring', ...page(() => import('@/pages/tracking/TrackingPage')) },
  { path: '/geofence', title: 'Geofencing', icon: ic(MapPin), resource: 'monitoring', ...page(() => import('@/pages/geofence/GeofencePage')) },
  { path: '/alert', title: 'Abnormal Alerts', icon: ic(AlertTriangle), resource: 'monitoring', ...page(() => import('@/pages/alert/AlertListPage')) },
  { path: '/distribution-maps', title: 'Abnormal Distribution Map', icon: ic(Locate), resource: 'monitoring', ...page(() => import('@/pages/distribution-maps/DistributionMapPage')) },
  { path: '/position-history', title: 'Position History', icon: ic(ChartNoAxesCombined), resource: 'monitoring', ...page(() => import('@/pages/position-history/PositionHistoryPage')) },
  { path: '/speed-per-segment', title: 'Speed Heatmap', icon: ic(CircleGauge), resource: 'monitoring', ...page(() => import('@/pages/speed-per-segment/SpeedPerSegmentPage')) },
  { path: '/fuel-history', title: 'Fuel History', icon: ic(Fuel), resource: 'monitoring', ...page(() => import('@/pages/fuel-history/FuelHistoryPage')) },
  { path: '/report', title: 'Report', icon: ic(FileText), resource: 'report', ...page(() => import('@/pages/report/ReportPage')) },
  { path: '/documents', title: 'Dokumen SOP', icon: ic(BookOpen), resource: 'documents', ...page(() => import('@/pages/documents/DocumentsPage')) },
  {
    key: 'upload',
    title: 'Record / Upload',
    icon: ic(Upload),
    children: [
      {
        key: 'transaction',
        title: 'Transaction',
        icon: ic(ArrowLeftRight),
        children: [
          { path: '/upload/daily-setting-operator', title: 'Setting Operator', icon: ic(UserPlus), resource: 'daily', ...page(() => import('@/pages/upload-data/daily-setting-operator/DailySettingOperatorPage')) },
          { path: '/upload/equipment-activation', title: 'Equipment Activation', icon: ic(Power), resource: 'daily', ...page(() => import('@/pages/upload-data/equipment-activation/EquipmentActivationPage')) },
          { path: '/upload/status-breakdown', title: 'Status Breakdown', icon: ic(BarChart), resource: 'daily', ...page(() => import('@/pages/upload-data/status-breakdown/StatusBreakdownPage')) },
          { path: '/upload/weighbridge', title: 'Weighbridge', icon: ic(Scale), resource: 'daily', ...page(() => import('@/pages/upload-data/weighbridge/WeighbridgePage')) },
          { path: '/upload/stock-emplacement', title: 'Stock Emplacement', icon: ic(Boxes), resource: 'daily', ...page(() => import('@/pages/upload-data/stock-emplacement/StockEmplacementPage')) },
          { path: '/upload/time-sheet', title: 'Time Sheet', icon: ic(ClipboardClock), resource: 'daily', ...page(() => import('@/pages/upload-data/time-sheet/TimeSheetPage')) },
          { path: '/upload/ritase', title: 'Ritase', icon: ic(Repeat), resource: 'daily', ...page(() => import('@/pages/upload-data/ritase/RitasePage')) },
          { path: '/upload/wh-notification', title: 'WH & Notification', icon: ic(Hourglass), resource: 'daily', ...page(() => import('@/pages/upload-data/wh-notification/WhNotificationPage')) },
          { path: '/upload/rain-slippery', title: 'Rain & Slippery', icon: ic(CloudRain), resource: 'daily', ...page(() => import('@/pages/upload-data/rain-slippery/RainSlipperyPage')) },
        ],
      },
      {
        key: 'plan',
        title: 'Plan',
        icon: ic(CalendarRange),
        children: [
          { path: '/upload/plan-yearly', title: 'Plan Yearly', icon: ic(CalendarDays), resource: 'daily', ...page(() => import('@/pages/upload-data/plan-yearly/PlanYearlyPage')) },
          { path: '/upload/plan-monthly', title: 'Plan Monthly', icon: ic(Calendar), resource: 'daily', ...page(() => import('@/pages/upload-data/plan-monthly/PlanMonthlyPage')) },
        ],
      },
    ],
  },
  {
    key: 'master',
    title: 'System Data',
    icon: ic(Sliders),
    children: [
      { path: '/master/user', title: 'User', icon: ic(SquareUser), resource: 'master', ...page(() => import('@/pages/master/user/UserPage')) },
      { path: '/master/company', title: 'Company', icon: ic(Building), resource: 'organization', ...page(() => import('@/pages/master/company/CompanyPage')) },
      { path: '/master/project', title: 'Project', icon: ic(Landmark), resource: 'organization', ...page(() => import('@/pages/master/project/ProjectPage')) },
      { path: '/master/equipment', title: 'Equipment', icon: ic(Truck), resource: 'master', ...page(() => import('@/pages/master/equipment/EquipmentPage')) },
      { path: '/master/device', title: 'Device', icon: ic(RadioReceiver), resource: 'master', ...page(() => import('@/pages/master/device/DevicePage')) },
      { path: '/master/map-area', title: 'Map Area', icon: ic(Layers), resource: 'master', ...page(() => import('@/pages/master/map-area/MapAreaPage')) },
      { path: '/master/map-editor', title: 'Map Editor', icon: ic(PenTool), resource: 'master', ...page(() => import('@/pages/master/map-editor/MapEditorPage')) },
      { path: '/fuel-calibration', title: 'Fuel Calibration', icon: ic(Fuel), resource: 'master', ...page(() => import('@/pages/fuel-calibration/FuelCalibrationPage')) },
      { path: '/master/shift', title: 'Shift', icon: ic(Clock), resource: 'master', ...page(() => import('@/pages/master/shift/ShiftPage')) },
      { path: '/master/alert-category', title: 'Alert Category', icon: ic(OctagonAlert), resource: 'master', ...page(() => import('@/pages/master/alert-category/AlertCategoryPage')) },
      { path: '/master/alert-rule', title: 'Alert Rule', icon: ic(BellRing), resource: 'master', ...page(() => import('@/pages/master/alert-rule/AlertRulePage')) },
      { path: '/master/report-schedule', title: 'Report Schedule', icon: ic(CalendarClock), resource: 'master', ...page(() => import('@/pages/master/report-schedule/ReportSchedulePage')) },
    ],
  },
]

/** Halaman di luar menu. */
const hidden = (path: string, title: string, resource: Resource, p: { page: Page }): PageRoute => ({ path, title, resource, menu: false, ...p })
const EXTRA: PageRoute[] = [
  hidden('/profile', 'Profil Saya', 'monitoring', page(() => import('@/pages/profile/ProfilePage'))),
  // alert per jenis (FMS lama: route ada, menu disembunyikan)
  hidden('/fuel-alert', 'Fuel Alert', 'monitoring', page(() => import('@/pages/alert/fuel/FuelAlertPage'))),
  hidden('/overspeed', 'Overspeed Alert', 'monitoring', page(() => import('@/pages/alert/overspeed/OverspeedAlertPage'))),
  hidden('/underspeed', 'Underspeed Alert', 'monitoring', page(() => import('@/pages/alert/underspeed/UnderspeedAlertPage'))),
  hidden('/offtrack', 'Off Track Alert', 'monitoring', page(() => import('@/pages/alert/off-track/OffTrackAlertPage'))),
  hidden('/graphic', 'Equipment Graphic', 'monitoring', page(() => import('@/pages/graphic/GraphicPage'))),
  // laporan (dibuka dari halaman Report)
  hidden('/report/report-alert-summary', 'Alert Summary', 'report', page(() => import('@/pages/report/report-alert-summary/ReportAlertSummaryPage'))),
  hidden('/report/report-executive-emplacement', 'Executive Summary Report Emplacement', 'report', page(() => import('@/pages/report/report-executive-emplacement/ReportExecutiveEmplacementPage'))),
  hidden('/report/report-executive-hauling', 'Executive Summary Report Hauling', 'report', page(() => import('@/pages/report/report-executive-hauling/ReportExecutiveHaulingPage'))),
  hidden('/report/report-equipment-logs', 'Asset History', 'report', page(() => import('@/pages/report/report-equipment-logs/ReportEquipmentLogsPage'))),
  hidden('/report/report-fuel-history', 'Fuel History Report', 'report', page(() => import('@/pages/report/report-fuel-history/ReportFuelHistoryPage'))),
  hidden('/report/report-ritase-daily', 'Ritase Daily', 'report', page(() => import('@/pages/report/report-ritase-daily/ReportRitaseDailyPage'))),
  hidden('/report/report-payload-daily', 'Payload Daily', 'report', page(() => import('@/pages/report/report-payload-daily/ReportPayloadDailyPage'))),
  hidden('/report/report-production-daily', 'Production Daily', 'report', page(() => import('@/pages/report/report-production-daily/ReportProductionDailyPage'))),
  hidden('/report/report-breakdown-daily', 'Breakdown Daily', 'report', page(() => import('@/pages/report/report-breakdown-daily/ReportBreakdownDailyPage'))),
  hidden('/report/report-pa-daily', 'PA Daily', 'report', page(() => import('@/pages/report/report-pa-daily/ReportPADailyPage'))),
  hidden('/report/equipment-shift-performance', 'Equipment Shift Performance', 'master', page(() => import('@/pages/report/EquipmentShiftPerformancePage'))),
  hidden('/report/playback-unit-history', 'Playback Unit History', 'report', page(() => import('@/pages/report/PlaybackUnitHistoryPage'))),
]

/** Semua halaman (rata) untuk router & pencarian judul. */
export const ROUTES: PageRoute[] = [...MENU.flatMap(pagesOf), ...EXTRA]

export const HOME = '/tracking'

export const routeOf = (pathname: string): PageRoute | undefined => ROUTES.find((r) => r.path === pathname)
