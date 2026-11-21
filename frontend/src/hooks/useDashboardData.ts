import { useMemo } from 'react'

import type { AlertHourRow, DashboardKpi, DashboardUnitRow, UnitListItem, TimeUsage } from '@/types/dashboard.types'
import type { Slot } from '@/types/telemetry.types'
import { alertTypeLabel } from '@/utils/colors'
import { pa as availability } from '@/utils/reports'
import { shiftHours, toLocal } from '@/utils/time'

import { useAlerts } from './useAlerts'
import { useCodeOf } from './useCodeOf'
import { useBreakdownDaily, useWeighbridgeDaily } from './useDailyReport'
import { useEquipmentBreakdowns } from './useEquipmentBreakdowns'
import { useEquipmentList } from './useEquipments'
import { useFuelEvents } from './useFuelEvents'
import { useLiveStates } from './useLiveStates'
import { useShifts } from './useShifts'
import { useActivity } from './useTelemetry'

const round = (v: number, d = 1) => Math.round(v * 10 ** d) / 10 ** d
const sum = (vs: number[]) => vs.reduce((s, v) => s + v, 0)

/** Unit milik sendiri. Kode berawalan DT8 (mis. DT809093) = unit mitra, tidak dihitung di produksi. */
export const isOwnUnit = (code: string) => !/^DT8/i.test(code)

/** Tipe unit yang ditampilkan di Bagian 1 Dashboard (urutan tampil). */
export const FLEET_TYPES = ['Dump Truck', 'Water Truck', 'Fuel Truck'] as const
export type FleetTypeRow = { type: string; total: number; online: number; running: number; idle: number; stop: number; breakdown: number }

/** Semua angka Dashboard untuk satu tanggal operasional + shift (kosong = semua shift). */
export function useDashboardData(slot: Slot) {
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  const shift = shifts.find((s) => s.shift_code === slot.shift)
  const shiftId = shift?.id
  const { data: states = [] } = useLiveStates()
  const { data: equipments = [] } = useEquipmentList(true)
  const activity = useActivity({ date: slot.date, shift: slot.shift })
  const alerts = useAlerts({ date: slot.date, shift: slot.shift })
  const fuel = useFuelEvents({ date: slot.date, shift: slot.shift })
  const wb = useWeighbridgeDaily({ from: slot.date, to: slot.date, shiftId })
  const bd = useBreakdownDaily({ from: slot.date, to: slot.date, shiftId })
  const breakdowns = useEquipmentBreakdowns(slot.date, shiftId)

  /** Jam tersedia per unit: panjang shift; semua shift = 24 jam. */
  const hoursAvailable = shift ? shiftHours(shift).length : 24

  const byStatus = useMemo(() => {
    const m: Record<string, number> = {}
    for (const s of states) {
      const k = s.is_breakdown ? 'breakdown' : (s.equipment_status ?? 'offline')
      m[k] = (m[k] ?? 0) + 1
    }
    return m
  }, [states])

  /** Kondisi armada dikelompokkan per tipe unit (status live; breakdown menimpa status operasi). */
  const byType = useMemo<FleetTypeRow[]>(() => {
    const typeOf = new Map(equipments.map((e) => [e.id, e.equipment_type]))
    const m = new Map<string, FleetTypeRow>(FLEET_TYPES.map((t) => [t, { type: t, total: 0, online: 0, running: 0, idle: 0, stop: 0, breakdown: 0 }]))
    for (const s of states) {
      const row = m.get(typeOf.get(s.equipment_id) ?? '')
      if (!row) continue
      const st = s.is_breakdown ? 'breakdown' : (s.equipment_status ?? 'offline')
      row.total++
      if (s.equipment_status && s.equipment_status !== 'offline') row.online++
      if (st === 'running' || st === 'idle' || st === 'stop' || st === 'breakdown') row[st]++
    }
    return [...m.values()]
  }, [states, equipments])

  /**
   * Unit aktif yang tidak ada di telemetry_events pada tanggal + shift ini: aktivitas dihitung dari telemetry_events,
   * jadi unit tanpa baris aktivitas = tidak punya satu pun data. Status = kondisi live terakhir yang diketahui.
   */
  const noTelemetry = useMemo<UnitListItem[]>(() => {
    if (!activity.isSuccess) return []
    const seen = new Set((activity.data?.data ?? []).map((a) => a.equipment_id))
    const stateOf = new Map(states.map((s) => [s.equipment_id, s]))
    return equipments
      .filter((e) => e.is_active && !seen.has(e.id))
      .map((e) => {
        const s = stateOf.get(e.id)
        return { id: e.id, code: e.equipment_code, type: e.equipment_type, status: s ? (s.is_breakdown ? 'breakdown' : (s.equipment_status ?? 'offline')) : null, lastEventAt: s?.event_time ?? null }
      })
      .sort((a, b) => a.code.localeCompare(b.code))
  }, [activity.isSuccess, activity.data, equipments, states])

  /** Daftar unit di balik angka Unit terpantau / Online / Offline (kondisi live saat ini, sama dengan dasar hitung KPI). */
  const unitLists = useMemo(() => {
    const info = new Map(equipments.map((e) => [e.id, e]))
    const all: UnitListItem[] = states.map((s) => ({
      id: s.equipment_id,
      code: codeOf(s.equipment_id),
      type: info.get(s.equipment_id)?.equipment_type ?? null,
      status: s.is_breakdown ? 'breakdown' : (s.equipment_status ?? 'offline'),
      lastEventAt: s.event_time,
    }))
    all.sort((a, b) => a.code.localeCompare(b.code))
    const stateOnline = new Set(states.filter((s) => s.equipment_status && s.equipment_status !== 'offline').map((s) => s.equipment_id))
    return {
      monitored: all,
      // sama dengan KPI: online = equipment_status bukan offline (breakdown tidak mengubah hitungan ini)
      online: all.filter((u) => stateOnline.has(u.id)),
      offline: all.filter((u) => !stateOnline.has(u.id)),
    }
  }, [states, equipments, codeOf])

  const alertList = useMemo(() => alerts.data?.data ?? [], [alerts.data])

  const rows = useMemo<DashboardUnitRow[]>(() => {
    const m = new Map<string, DashboardUnitRow>()
    const get = (code: string) =>
      m.get(code) ??
      m.set(code, { code, status: 'offline', pa: 100, running: 0, idle: 0, avg: 0, max: 0, mileage: 0, fuelUsed: 0, burn: 0, ritase: 0, production: 0, breakdown: 0, alerts: 0 }).get(code)!
    for (const s of states) get(codeOf(s.equipment_id)).status = s.is_breakdown ? 'breakdown' : (s.equipment_status ?? 'offline')
    for (const a of activity.data?.data ?? []) {
      Object.assign(get(codeOf(a.equipment_id)), { running: a.running_hours, idle: a.idle_hours, avg: a.avg_speed, max: a.max_speed, mileage: a.mileage_km, fuelUsed: a.fuel_decrease_liter, burn: a.fuel_burn_km_per_liter })
    }
    for (const c of wb.data ?? []) Object.assign(get(c.equipment_code), { ritase: c.count, production: round(c.sum_kg / 1000, 2) })
    for (const c of bd.data ?? []) get(c.equipment_code).breakdown = round(c.sum_hours, 2)
    for (const a of alertList) get(codeOf(a.equipment_id)).alerts++
    const out = [...m.values()]
    for (const r of out) r.pa = availability(r.breakdown, hoursAvailable)
    return out.sort((a, b) => a.code.localeCompare(b.code))
  }, [states, activity.data, wb.data, bd.data, alertList, codeOf, hoursAvailable])

  const ownRows = useMemo(() => rows.filter((r) => isOwnUnit(r.code)), [rows])

  const kpi = useMemo<DashboardKpi>(() => {
    const units = states.length || rows.length
    const online = states.filter((s) => s.equipment_status && s.equipment_status !== 'offline').length
    const running = sum(rows.map((r) => r.running))
    const breakdownHours = sum(rows.map((r) => r.breakdown))
    const own = rows.filter((r) => isOwnUnit(r.code))
    const ritase = sum(own.map((r) => r.ritase))
    const tonnes = sum(own.map((r) => r.production))
    const moving = rows.filter((r) => r.running > 0)
    const burnRows = rows.filter((r) => r.burn > 0)
    return {
      units,
      online,
      offline: units - online,
      running: byStatus.running ?? 0,
      ritase,
      tonnes: round(tonnes, 1),
      payload: ritase ? round(tonnes / ritase, 1) : 0,
      pa: units ? availability(breakdownHours / units, hoursAvailable) : 0,
      utilization: units ? round((running / (units * hoursAvailable)) * 100, 0) : 0,
      mileage: round(sum(rows.map((r) => r.mileage)), 0),
      avgSpeed: moving.length ? round(sum(moving.map((r) => r.avg)) / moving.length, 1) : 0,
      fuelUsed: round(sum(rows.map((r) => r.fuelUsed)), 0),
      fuelFilled: round(sum((fuel.data?.data ?? []).filter((f) => f.event_type === 'increase').map((f) => f.fuel_delta_liter)), 0),
      burn: burnRows.length ? round(sum(burnRows.map((r) => r.burn)) / burnRows.length, 2) : 0,
      alerts: alertList.length,
      breakdownHours: round(breakdownHours, 1),
    }
  }, [states, rows, byStatus, fuel.data, alertList, hoursAvailable])

  const timeUsage = useMemo<TimeUsage>(() => {
    const running = round(sum(rows.map((r) => r.running)), 1)
    const idle = round(sum(rows.map((r) => r.idle)), 1)
    const breakdown = round(sum(rows.map((r) => r.breakdown)), 1)
    const total = (states.length || rows.length) * hoursAvailable
    return { running, idle, breakdown, other: Math.max(round(total - running - idle - breakdown, 1), 0) }
  }, [rows, states.length, hoursAvailable])

  const alertByType = useMemo(
    () => Object.keys(alertTypeLabel).map((type) => ({ type, count: alertList.filter((a) => a.alert_type === type).length })),
    [alertList],
  )

  /** Jam-jam shift (semua shift = 24 jam) dengan jumlah alert per jenis. */
  const alertByHour = useMemo<AlertHourRow[]>(() => {
    const hours = shift ? shiftHours(shift) : Array.from({ length: 24 }, (_, i) => i)
    return hours.map((h) => ({
      hour: `${String(h).padStart(2, '0')}:00`,
      counts: Object.fromEntries(Object.keys(alertTypeLabel).map((t) => [t, alertList.filter((a) => a.alert_type === t && toLocal(a.started_at).hour() === h).length])),
    }))
  }, [alertList, shift])

  const recentAlerts = useMemo(
    () => [...alertList].sort((a, b) => b.started_at.localeCompare(a.started_at)).slice(0, 20).map((a) => ({ ...a, code: codeOf(a.equipment_id) })),
    [alertList, codeOf],
  )

  return {
    shift,
    hoursAvailable,
    byStatus,
    byType,
    kpi,
    noTelemetry,
    unitLists,
    noTelemetryReady: activity.isSuccess,
    rows,
    ownRows,
    timeUsage,
    alertByType,
    alertByHour,
    recentAlerts,
    breakdowns: breakdowns.data ?? [],
    loading: activity.isFetching || wb.isFetching || bd.isFetching || alerts.isFetching,
  }
}
