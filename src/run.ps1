$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

# pastikan go di PATH (Windows)
$env:Path = "C:\Program Files\Go\bin;$env:Path"

Write-Host "== build engine ==" -ForegroundColor Cyan
Push-Location $root
# On Windows the executed binary is bin/gs.exe, so build to that exact name.
# Building to "bin/gs" would create a separate (no-extension) file and leave
# bin/gs.exe stale, which is a nasty trap.
go build -o bin/gs.exe ./cmd/garudashield
Pop-Location

Write-Host "== start origin (background) ==" -ForegroundColor Cyan
$origin = $null
$originDir = "C:\temp\gs-origin"
if (-not (Get-NetTCPConnection -LocalPort 3001 -State Listen -ErrorAction SilentlyContinue)) {
  if (-not (Test-Path $originDir)) { New-Item -ItemType Directory -Path $originDir -Force | Out-Null }
  if (-not (Test-Path "$originDir\index.html")) { Set-Content -Path "$originDir\index.html" -Value "<h1>GarudaShield origin ok</h1>" }
  $origin = Start-Process -FilePath "python" -ArgumentList @("-m", "http.server", "3001", "--directory", $originDir) -PassThru
  Start-Sleep -Seconds 1
}

Write-Host "== start engine (background) ==" -ForegroundColor Cyan
# Commercial control plane (PostgreSQL) env. Defaults target the local
# user-mode Postgres documented in log.md "Batch 33"; override any of them in
# your shell beforehand to point elsewhere. Without GS_DATABASE_URL the engine
# still starts, but /api/v1/* (tenant auth, license, members, audit) is not
# registered and the customer portal cannot sign in.
if (-not $env:GS_DATABASE_URL) { $env:GS_DATABASE_URL = "postgres://garuda:GarudaShield2026Pg@127.0.0.1:5432/garudashield?sslmode=disable" }
if (-not $env:GS_SESSION_SECRET) { $env:GS_SESSION_SECRET = "gsec-session-9f2c1a7b4e6d8f0a1b2c3d4e5f607182" }
if (-not $env:GS_LICENSE_SECRET) { $env:GS_LICENSE_SECRET = "gsec-license-A1B2C3D4E5F60718293a4b5c6d7e8f90" }
# GS_ADMIN_SECRET gates operator endpoints (/api/admin/tenant/onboard and
# /api/admin/license/*). Keep it identical to the dashboard's GS_API_KEY.
if (-not $env:GS_ADMIN_SECRET) { $env:GS_ADMIN_SECRET = "855d71cafc1438088ef25ac8f40e1028ab80a943685503a5e03a2c8ad741087a" }
$engine = Start-Process -FilePath "$root\bin\gs.exe" -ArgumentList @("-config", "$root\tools\attack-config.json") -WorkingDirectory $root -PassThru
Start-Sleep -Seconds 2

try {
  Write-Host "== start web (npm run dev) - Ctrl+C untuk berhenti ==" -ForegroundColor Cyan
  Set-Location "$root\dashboard"
  $env:GS_API_URL = "http://localhost:8080"
  if (-not $env:GS_API_KEY) { throw "GS_API_KEY belum di-set. Buat dengan: [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Max 256 })) lalu `$env:GS_API_KEY='<hasil>'" }
  if (-not (Test-Path node_modules)) { npm install }
  npm run dev
}
finally {
  Write-Host "== stopping engine ==" -ForegroundColor Cyan
  Stop-Process -Id $engine.Id -Force -ErrorAction SilentlyContinue
  if ($origin) { Stop-Process -Id $origin.Id -Force -ErrorAction SilentlyContinue }
}
