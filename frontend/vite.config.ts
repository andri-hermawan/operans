/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// Saat development, browser hanya bicara ke Vite (port 8881). Vite meneruskan:
//   /core/... → core-api (3346)   /iot/... dan /iot/ws/live → layanan IoT (3347)   /ai/... → AI Brain (3348)
// Di produksi, Nginx melakukan hal yang sama → tidak ada masalah CORS.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const core = env.CORE_API_URL || 'http://localhost:3346'
  const iot = env.IOT_API_URL || 'http://localhost:3347'
  const ai = env.AI_API_URL || 'http://127.0.0.1:3348'
  const base = `/${(env.APP_BASE_PATH || '/operans').replace(/^\/+|\/+$/g, '')}/`
  return {
    base, // aplikasi dibuka di http://localhost:PORT/operans/ (API tetap di /core, /iot, /ai)
    plugins: [react()],
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
