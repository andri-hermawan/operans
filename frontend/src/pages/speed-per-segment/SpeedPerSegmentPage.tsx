// Speed Heatmap — peta diwarnai pita kecepatan. Titik telemetry TIDAK digambar satu per satu: server meringkasnya
// menjadi kotak-kotak (ukuran mengikuti zoom), sehingga tetap ringan walau unit ratusan. Rata-rata kosong/bermuatan
// per segmen (label peta) tetap dihitung dari seluruh data telemetry lewat endpoint segment-speeds.
import { useMemo, useState } from 'react'
import { Empty, Switch, Typography } from 'antd'
import { Popup } from 'react-leaflet'

import type { HeatCell, MapView } from '@/types/telemetry.types'
import type { ZoneCollection } from '@/types/zone.types'
import { useHeatGrid } from '@/hooks/useHeatGrid'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { useSegmentSpeeds } from '@/hooks/useTelemetry'
import { useZones } from '@/hooks/useZones'
import { speedBands, speedColor } from '@/utils/colors'
import { formatNumber } from '@/utils/format'
import SlotFilters, { SlotSelect } from '@/components/filters/SlotFilters'
import BaseMap from '@/components/map/BaseMap'
import { Dot, FitTo, Legend, MapViewWatcher, ZoneLayer } from '@/components/map/layers'
import MapShell from '@/components/map/MapShell'
import InfoRows from '@/components/monitoring/InfoRows'
import ListHeader from '@/components/monitoring/ListHeader'
import styles from '@/components/monitoring/monitoring.module.css'
import PageHeader from '@/components/ui/PageHeader'

import HeatCellCard from './components/HeatCellCard'

const BAND_OPTIONS = speedBands.map((b, i) => ({ value: String(i), label: `${b.label} km/h` }))
const LIST_SIZE = 50
const cellKey = (c: HeatCell) => `${c.latitude},${c.longitude}`

export default function SpeedPerSegmentPage() {
  const { data: zones } = useZones()
  const { values, set, slot } = useSlotFilters({ band: undefined })
  const [view, setView] = useState<MapView>()
  const grid = useHeatGrid({ date: slot.date, shift: slot.shift }, view)
  const speeds = useSegmentSpeeds({ date: slot.date, shift: slot.shift })
  const [labels, setLabels] = useState(false)
  const [selected, setSelected] = useState<string>()

  const band = values.band != null ? speedBands[Number(values.band)] : undefined
  const min = band ? (speedBands[Number(values.band) - 1]?.max ?? -1) : -1
  const cells = useMemo(
    () => (grid.data?.data ?? []).filter((c) => !band || (c.avg_speed > min && c.avg_speed <= band.max)),
    [grid.data, band, min],
  )
  const top = useMemo(() => [...cells].sort((a, b) => b.points - a.points).slice(0, LIST_SIZE), [cells])
  const cellM = grid.cellM ?? 20
  const sel = cells.find((c) => cellKey(c) === selected)

  // nama area diberi rata-rata speed "S,E x / S,L y" (dari seluruh data, bukan dari kotak di layar)
  const labelled = useMemo<ZoneCollection | undefined>(() => {
    if (!zones) return undefined
    const m = new Map((speeds.data?.data ?? []).map((s) => [s.segment_name.toLowerCase(), s]))
    return {
      ...zones,
      features: zones.features.map((f) => {
        const s = m.get(f.properties.name.toLowerCase())
        return s ? { ...f, properties: { ...f.properties, name: `${f.properties.name} — S,E ${formatNumber(s.avg_speed_empty, 1)} / S,L ${formatNumber(s.avg_speed_loaded, 1)}` } } : f
      }),
    }
  }, [zones, speeds.data])

  const panel = (
    <>
      <SlotFilters asset={false} values={values} onChange={set} extra={<SlotSelect label="Speed" value={values.band} options={BAND_OPTIONS} onChange={(v) => set({ band: v })} />} />
      <label style={{ display: 'flex', gap: 8, alignItems: 'center', fontSize: 13 }}>
        <Switch size="small" checked={labels} onChange={setLabels} /> Tampilkan label speed segmen
      </label>
      <ListHeader title="Speed Heatmap List" count={cells.length} />
      {grid.data?.truncated && <Typography.Text type="warning">Kotak terlalu banyak — perbesar (zoom) peta untuk melihat lebih detail.</Typography.Text>}
      <div className={`${styles.list} ${styles.listBox}`}>
        {!top.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={grid.isFetching ? 'Memuat…' : 'Tidak ada data'} />}
        {top.map((c) => (
          <HeatCellCard key={cellKey(c)} cell={c} active={cellKey(c) === selected} onClick={() => setSelected(cellKey(c))} />
        ))}
        {cells.length > LIST_SIZE && <Typography.Text type="secondary">Menampilkan {LIST_SIZE} kotak paling sering dilalui dari {formatNumber(cells.length)} kotak.</Typography.Text>}
      </div>
    </>
  )

  return (
    <>
      <PageHeader title="Speed Heatmap" subtitle={`Titik dikelompokkan per kotak ±${cellM} m (mengikuti zoom peta)`} />
      <MapShell
        panel={panel}
        map={(open) => (
          <BaseMap resizeKey={open}>
            <ZoneLayer zones={labelled} labels={labels} />
            <FitTo area={zones} fitKey={`${slot.date}|${slot.shift}`} />
            <MapViewWatcher onChange={setView} />
            {cells.map((c) => (
              <Dot key={cellKey(c)} position={[c.latitude, c.longitude]} radius={5} color={speedColor(c.avg_speed)} selected={cellKey(c) === selected} onClick={() => setSelected(cellKey(c))} />
            ))}
            {sel && (
              <Popup position={[sel.latitude, sel.longitude]} minWidth={230} eventHandlers={{ remove: () => setSelected(undefined) }}>
                <InfoRows
                  title="Kotak heatmap"
                  d={{
                    speed: sel.avg_speed,
                    lat: sel.latitude,
                    lon: sel.longitude,
                    extra: [
                      { label: 'Maks', value: `${formatNumber(sel.max_speed, 1)} km/h` },
                      { label: 'Kosong', value: sel.avg_speed_empty == null ? '-' : `${formatNumber(sel.avg_speed_empty, 1)} km/h` },
                      { label: 'Bermuatan', value: sel.avg_speed_loaded == null ? '-' : `${formatNumber(sel.avg_speed_loaded, 1)} km/h` },
                      { label: 'Titik', value: formatNumber(sel.points) },
                      { label: 'Unit', value: sel.units },
                    ],
                  }}
                />
              </Popup>
            )}
            <Legend title="Speed (km/h)" items={speedBands.map((b) => ({ color: b.color, label: b.label }))} />
          </BaseMap>
        )}
      />
    </>
  )
}
