package domain

// HeatCell = satu kotak peta pada Speed Heatmap: semua titik telemetry yang jatuh di kotak itu diringkas menjadi satu
// baris (supaya peta tetap ringan walau ada ratusan unit). Rata-rata dihitung dari SEMUA titik di kotak, bukan sampel.
type HeatCell struct {
	Latitude  float64  `json:"latitude"` // titik tengah kotak
	Longitude float64  `json:"longitude"`
	Points    int      `json:"points"`           // jumlah titik telemetry di kotak
	Units     int      `json:"units"`            // jumlah unit berbeda yang melintas
	AvgSpeed  float64  `json:"avg_speed"`        // km/jam
	MaxSpeed  float64  `json:"max_speed"`        //
	AvgEmpty  *float64 `json:"avg_speed_empty"`  // hanya titik berstatus kosong
	AvgLoaded *float64 `json:"avg_speed_loaded"` // hanya titik berstatus bermuatan
}

// HeatGridQuery = parameter permintaan kotak heatmap.
type HeatGridQuery struct {
	CellMeters                     float64 // sisi kotak (meter)
	MinLon, MinLat, MaxLon, MaxLat float64 // batas peta yang terlihat
	HasBBox                        bool
	Limit                          int
}
