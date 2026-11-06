// Tema Ant Design — satu-satunya tempat pengaturan warna, huruf, sudut & ukuran komponen.
import type { ThemeConfig } from 'antd'

import { brand } from '@/utils/colors'

export const LAYOUT = {
  siderWidth: 260,
  siderCollapsed: 80,
  headerHeight: 60,
  contentPadding: 24,
  contentPaddingMobile: 12,
} as const

export const theme: ThemeConfig = {
  token: {
    colorPrimary: brand.primary,
    colorInfo: brand.info,
    colorSuccess: brand.success,
    colorWarning: brand.warning,
    colorError: brand.danger,
    colorTextBase: brand.text,
    colorBgLayout: brand.background,
    colorBorderSecondary: brand.border,
    borderRadius: 8,
    fontFamily: "Inter, 'Segoe UI', system-ui, -apple-system, Roboto, sans-serif",
    fontSize: 14,
  },
  components: {
    Layout: { headerBg: brand.surface, siderBg: brand.surface, bodyBg: brand.background, headerPadding: '0 24px' },
    Menu: {
      itemBg: 'transparent',
      itemSelectedBg: '#e8f0f9',
      itemSelectedColor: brand.primary,
      itemHoverBg: '#f1f5f9',
      itemBorderRadius: 8,
      itemMarginInline: 8,
      subMenuItemBg: 'transparent',
      iconSize: 18,
    },
    Card: { headerBg: brand.surface, headerFontSize: 15 },
    Table: { headerBg: '#f1f5f9', headerColor: brand.text, rowHoverBg: '#f8fafc', cellPaddingBlockSM: 6 },
    Button: { fontWeight: 500 },
  },
}
