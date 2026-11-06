// Daftar proses OPERANS untuk PM2 (dipakai di server).
//
// Persiapan sekali / setiap habis git pull (jalankan dari folder ini):
//   cd services\core-api-go       && go build -o core-api.exe ./cmd/api
//   cd services\iot-ingestion-go  && go build -o iot.exe ./cmd/iot
//   cd frontend                   && npm install && npm run build
//
// Menjalankan:   pm2 start ecosystem.config.cjs
// Setelah update: pm2 restart ecosystem.config.cjs
// Menyimpan agar bisa dipulihkan setelah server restart:   pm2 save
//
// Tiap proses membaca file .env di foldernya sendiri (cwd), jadi path di bawah tidak perlu diubah
// walau project dipindah ke folder lain: semuanya relatif terhadap file ini.
const path = require('node:path')

const root = __dirname
const at = (...p) => path.join(root, ...p)

module.exports = {
  apps: [
    {
      name: 'operans-core', // core-api, port 3346
      script: at('services', 'core-api-go', 'core-api.exe'),
      cwd: at('services', 'core-api-go'),
    },
    {
      name: 'operans-iot', // IoT: HTTP 3347 + TCP alat GPS 5550
      script: at('services', 'iot-ingestion-go', 'iot.exe'),
      cwd: at('services', 'iot-ingestion-go'),
    },
    {
      name: 'operans-web', // frontend hasil build (dist), port 8881, buka di /operans/
      script: at('frontend', 'node_modules', 'vite', 'bin', 'vite.js'),
      args: 'preview --host 0.0.0.0 --port 8881',
      cwd: at('frontend'),
    },
    // AI Brain (port 3348) — hapus tanda komentar bila dipakai di server ini.
    // Persiapan sekali ada di services\ai-brain-py\README.md (buat .venv dan pasang paket).
    // {
    //   name: 'operans-ai',
    //   script: at('services', 'ai-brain-py', '.venv', 'Scripts', 'uvicorn.exe'),
    //   args: 'app.main:build_app --factory --port 3348',
    //   cwd: at('services', 'ai-brain-py'),
    // },
  ],
}
