// "Tampilan awal" peta: FitTo mencatat batas yang sedang dipakai, tombol Reset View terbang kembali ke sana.
import { createContext, useContext, type RefObject } from 'react'
import type { LatLngBoundsExpression } from 'leaflet'

export interface HomeView {
  bounds: LatLngBoundsExpression
  padding: number
}

export const MapHomeContext = createContext<RefObject<HomeView | undefined> | null>(null)

export const useMapHome = () => useContext(MapHomeContext)
