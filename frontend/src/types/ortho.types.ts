// Citra orthophoto (tile XYZ) yang dilayani sebagai berkas statis di /ortho/<id>/{z}/{x}/{y}.<ext>.

export interface OrthoVersion {
  /** Nama folder tile, mis. "R1". */
  id: string
  name: string
  /** Tanggal pemotretan (teks bebas, mis. "2026-11") atau null bila belum diketahui. */
  date: string | null
  /** false = tidak menyala otomatis (bawaan: semua versi menyala). */
  default?: boolean
  /** [[lat selatan, lon barat], [lat utara, lon timur]] */
  bounds: [[number, number], [number, number]]
  min_zoom: number
  max_zoom: number
  ext: string
}

export interface OrthoIndex {
  versions: OrthoVersion[]
}
