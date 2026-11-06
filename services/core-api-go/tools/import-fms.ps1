# B8 - pindahkan data CHR dari fms_lite (HANYA DIBACA) ke simha. Aman dijalankan berulang.
# Jalankan dari folder core-api-go. Butuh .env.migrate (simha, core_migrator) dan .env aplikasi lama.
#
#   powershell -ExecutionPolicy Bypass -File tools\import-fms.ps1 [-Legacy "<folder fms-backend>"]

param([string]$Legacy = "C:\Users\ITHO-OPS-SYS-DEV\Desktop\All Project\FMS\BACKEND\fms-backend")

$src = (Select-String -Path (Join-Path $Legacy ".env") -Pattern '^DATABASE_URL=' | Select-Object -First 1).Line
$env:FMS_DATABASE_URL = ($src -replace '^DATABASE_URL=', '').Trim('"') -replace '\?schema=.*$', ''
$env:TARGET_DATABASE_URL = (Select-String -Path .env.migrate -Pattern '^MIGRATE_DATABASE_URL=').Line -replace '^MIGRATE_DATABASE_URL=', ''
try { go run ./tools/import-fms } finally { Remove-Item Env:FMS_DATABASE_URL, Env:TARGET_DATABASE_URL -ErrorAction SilentlyContinue }
