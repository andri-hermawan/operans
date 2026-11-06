import dayjs from 'dayjs'

import { formatPlainDate } from '@/utils/format'

/** H-1 dari tanggal operasional (tanggal data laporan Executive Summary). */
export const dayBefore = (date: string) => dayjs(date).subtract(1, 'day').format('YYYY-MM-DD')

/** Teks tanggal untuk kop laporan Executive: data sampai H-1 dari tanggal operasional. */
export const executiveSubtitle = (date: string) => `Tanggal operasional ${formatPlainDate(date)} · data sampai ${formatPlainDate(dayBefore(date))} (H-1)`
