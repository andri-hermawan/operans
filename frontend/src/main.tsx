import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'leaflet/dist/leaflet.css'
import './index.css'
import { brand } from '@/utils/colors'
import App from './App'
import { adoptPrintToken } from '@/utils/printToken'

// Warna merek → variabel CSS (--simha-primary, ...), agar file .css memakai sumber warna yang sama dengan tema.
for (const [key, value] of Object.entries(brand)) {
  document.documentElement.style.setProperty(`--simha-${key.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`)}`, value)
}

adoptPrintToken() // halaman cetak PDF (jadwal laporan)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
