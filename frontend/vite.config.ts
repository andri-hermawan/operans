/// <reference types="vitest/config" />
import { createReadStream, statSync } from 'node:fs'
import { extname, join, normalize, resolve, sep } from 'node:path'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'

// Saat development, browser hanya bicara ke Vite (port 8881). Vite meneruskan:
//   /core/... → core-api (3346)   /iot/... dan /iot/ws/live → layanan IoT (3347)   /ai/... → AI Brain (3348)
// Di produksi, Nginx melakukan hal yang sama → tidak ada masalah CORS.
// Alamat tanpa garis miring (/operans) disamakan dengan /operans/ — tanpa ini Vite hanya menampilkan petunjuk
// "did you mean to visit /operans/". Dipakai di dev server dan di `vite preview` (server produksi).
function trailingSlashBase(base: string): Plugin {
  const bare = base.replace(/\/$/, '')
  const rewrite = (req: { url?: string }, _res: unknown, next: () => void) => {
    const url = req.url ?? ''
    if (url === bare || url.startsWith(`${bare}?`)) req.url = `${base}${url.slice(bare.length)}`
    next()
  }
  return {
    name: 'trailing-slash-base',
    configureServer: (server) => void server.middlewares.use(rewrite), // tanpa return → dipasang paling awal
    configurePreviewServer: (server) => void server.middlewares.use(rewrite),
  }
}

// Citra ortho: tile statis di folder `operans/ortho/tiles` (hasil `gdal raster tile`, tidak masuk Git) disajikan di
// /ortho/<versi>/{z}/{x}/{y}.webp dan /ortho/index.json. Di produksi Nginx menyajikan folder yang sama (lihat README).
// Letak folder bisa diganti dengan ORTHO_DIR di .env. Jalur dijaga agar tidak keluar dari folder tile.
const ORTHO_TYPES: Record<string, string> = { '.webp': 'image/webp', '.png': 'image/png', '.jpg': 'image/jpeg', '.json': 'application/json' }
function serveOrtho(dir: string): Plugin {
  const root = resolve(dir)
  const handle = (req: { url?: string }, res: import('node:http').ServerResponse, next: () => void) => {
    const url = (req.url ?? '').split('?')[0]
    if (!url.startsWith('/ortho/')) return next()
    const file = normalize(join(root, decodeURIComponent(url.slice('/ortho/'.length))))
    const type = ORTHO_TYPES[extname(file).toLowerCase()]
    let ok = file.startsWith(root + sep) && !!type
    if (ok) {
      try {
        ok = statSync(file).isFile()
      } catch {
        ok = false
      }
    }
    if (!ok) {
      res.statusCode = 404 // tile di luar area foto memang tidak ada
      return void res.end()
    }
    res.setHeader('Content-Type', type)
    res.setHeader('Cache-Control', type === 'application/json' ? 'no-cache' : 'public, max-age=86400')
    createReadStream(file).pipe(res)
  }
  return {
    name: 'serve-ortho',
    configureServer: (server) => void server.middlewares.use(handle),
    configurePreviewServer: (server) => void server.middlewares.use(handle),
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const core = env.CORE_API_URL || 'http://localhost:3346'
  const iot = env.IOT_API_URL || 'http://localhost:3347'
  const ai = env.AI_API_URL || 'http://127.0.0.1:3348'
  const base = `/${(env.APP_BASE_PATH || '/operans').replace(/^\/+|\/+$/g, '')}/`
  return {
    base, // aplikasi dibuka di http://localhost:PORT/operans/ (API tetap di /core, /iot, /ai)
    plugins: [react(), trailingSlashBase(base), serveOrtho(env.ORTHO_DIR || '../ortho/tiles')],
    resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
    server: {
      host: env.FRONTEND_HOST || '0.0.0.0', // 0.0.0.0 = bisa dibuka dari komputer lain; "localhost" = hanya server ini
      port: Number(env.FRONTEND_PORT) || 8881,
      proxy: {
        '/core': { target: core, changeOrigin: true, rewrite: (p) => p.replace(/^\/core/, '') },
        '/iot': { target: iot, changeOrigin: true, ws: true, rewrite: (p) => p.replace(/^\/iot/, '') },
        '/ai': { target: ai, changeOrigin: true, rewrite: (p) => p.replace(/^\/ai/, '') },
      },
    },
    // antd + react ±1 MB sebelum gzip (±320 kB gzip); halaman berat (peta, grafik, Excel) dimuat terpisah
    build: { chunkSizeWarningLimit: 1200 },
    test: { environment: 'node', include: ['src/**/*.test.ts'] },
  }
})
