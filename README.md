# GarudaShield

GarudaShield adalah layanan proteksi **DDoS** dan **WAF (Web Application Firewall)**
untuk website, UMKM, dan tim IT di Indonesia. Pengguna mendaftar, menambahkan domain
yang ingin dilindungi, lalu memantau dan mengatur proteksi dari satu dashboard.
Repositori ini berisi website GarudaShield (landing, harga, registrasi/login, dan
portal pelanggan beserta dashboard pemantauan).

## Alur Inti

1. Pengguna membuka halaman pendaftaran dan membuat akun; sistem otomatis memberi
   paket **Free** setelah registrasi berhasil.
2. Pengguna login dan masuk ke portal/dashboard GarudaShield.
3. Pengguna menambahkan domain website yang ingin dilindungi. Input divalidasi
   (kosong atau format salah ditolak dengan pesan jelas).
4. Pengguna mengaktifkan proteksi: mengikuti petunjuk konfigurasi/verifikasi domain
   sesuai paket yang aktif.
5. Pengguna memantau status proteksi, lalu lintas, dan aktivitas mencurigakan secara
   real-time, serta mengatur proteksi per domain.
6. Pengguna dapat meningkatkan paket ke **Pro** (Rp499.000/bln, atau tahunan dengan
   diskon 15%) atau menghubungi tim untuk **Enterprise** (multi-node, SLA, on-prem).

## Cara Menjalankan (How to Run)

### Prasyarat

| Komponen | Versi | Untuk |
|---|---|---|
| Node.js | 20+ (diuji v24.14.1) | Website & dashboard |
| Go | 1.24+ (opsional) | Engine proteksi; tidak perlu bila hanya melihat antarmuka |
| PostgreSQL | 16 (opsional) | Portal pelanggan (registrasi/login). Tanpa ini engine tetap jalan (mode degradasi) |
| Docker | opsional | Redis & PostgreSQL |
| Python 3 | opsional | Hanya untuk origin dummy `run.ps1` |
| Browser | Chrome / Firefox | — |

### Langkah

1. Clone repositori, lalu masuk ke foldernya:
   ```
   git clone https://github.com/IbnXhiper/steampreneur-garudashield.git
   cd steampreneur-garudashield
   ```
2. Install dependensi website dan jalankan (dari folder `dashboard`):
   ```
   cd dashboard
   npm install
   npm run dev
   ```
   Website tersedia di http://localhost:3000
3. (Opsional) Jalankan engine proteksi sekaligus dari folder root:
   ```
   .\run.ps1      # Windows (PowerShell)
   ./run.sh       # Linux / macOS / WSL
   ```
   > **Windows:** set `$env:GS_API_KEY` lebih dulu; `run.ps1` juga butuh **Python
   > asli** untuk origin dummy (`python -m http.server`). Bila `npm` gagal karena
   > Execution Policy, pakai `npm.cmd` atau `cmd /c npm ...`.
4. Buka http://localhost:3000 di browser.

### Database (opsional, untuk portal pelanggan)

Engine boot tanpa PostgreSQL, tetapi route `/api/v1/*` (registrasi/login pelanggan)
tidak terdaftar. Untuk mencoba portal, sediakan PostgreSQL 16 lalu set
`GS_DATABASE_URL` (`postgres://garuda:<password>@127.0.0.1:5432/garudashield?sslmode=disable`);
migrasi tabel dijalankan otomatis saat boot. Langkah lengkap ada di
[`docs/LOCAL_SETUP_WINDOWS.md`](docs/LOCAL_SETUP_WINDOWS.md).

### Akun atau data contoh

| Peran | Username | Password |
|---|---|---|
| Pelanggan (paket Free) | demo@garudashield.test | DemoGaruda2026! |

Pendaftaran dapat dilakukan sendiri dari portal: buka http://localhost:3000/portal,
klik **"Daftar gratis"**, lalu isi email & password (minimal 12 karakter). Setiap
akun baru otomatis mendapat paket Free (maksimal 1 domain) — ditetapkan oleh
backend, bukan dari input pengguna.

### Verifikasi cepat

```
go test ./internal/... ./cmd/... -count=1     # semua unit test lulus
curl http://localhost:8080/health             # 200 (bila engine jalan)
curl http://localhost:8080/api/status         # 200 (endpoint publik)
```
Buka http://localhost:3000 — halaman landing memuat (200) dan `/login` menerima
admin key.

### Jika terjadi masalah

- **Port 3000 atau 8080 sudah dipakai:** hentikan proses lain atau ubah port
  (`npm run dev -- -p 3001`).
- **`npm install` gagal / "running scripts is disabled":** pastikan Node.js 20+
  (`node -v`); bila `npm.ps1` diblokir Execution Policy, pakai `npm.cmd` atau
  `cmd /c npm install`.
- **`python` tidak ditemukan saat `run.ps1`:** instal Python dari python.org, atau
  jalankan engine manual (lihat panduan lengkap).
- **`/api/v1/*` balas 404:** engine boot tanpa PostgreSQL — set `GS_DATABASE_URL`
  lalu restart engine.
- **Redis/PostgreSQL tidak tersedia:** tidak masalah — aplikasi otomatis memakai
  penyimpanan lokal (degradasi graceful).
- **Data hilang setelah restart:** pastikan tidak menjalankan mode incognito dan
  cookie situs diizinkan.

## Tangkapan Layar Alur Inti

Minimal satu tangkapan layar untuk setiap langkah alur inti, termasuk hasil change
request. Gambar disimpan di `docs/bukti/`.

1. Langkah 1 — Registrasi (otomatis Free): ![Registrasi](docs/bukti/langkah-1.png)
2. Langkah 2 — Login: ![Login](docs/bukti/langkah-2.png)
3. Langkah 3 — Tambah domain: ![Tambah domain](docs/bukti/langkah-3.png)
4. Langkah 4 — Aktivasi proteksi: ![Aktivasi](docs/bukti/langkah-4.png)
5. Langkah 5 — Pantau dashboard: ![Dashboard](docs/bukti/langkah-5.png)
6. Langkah 6 — Upgrade paket Pro: ![Upgrade](docs/bukti/langkah-6.png)
7. Change request — pencarian/filter/ekspor CSV: ![Change request](docs/bukti/change-request.png)

## Cara Menjalankan Pengujian

Unit test engine (Go):
```
go test ./internal/... ./cmd/... -count=1
```
Pemeriksaan build website:
```
cd dashboard && npm run build
```
Hasil yang diharapkan: semua test lulus.

## Fitur

- Registrasi & login; akun baru otomatis mendapat paket Free.
- Manajemen domain: tambah domain dengan validasi format dan pesan kesalahan jelas.
- Batas paket: Free maksimal 1 domain, dengan informasi upgrade saat melebihi batas.
- Konfigurasi proteksi per domain (subset: mode, budget, L7, WAF, rate limit, GeoIP).
- Dashboard pemantauan trafik & ancaman real-time (polling/SSE).
- Halaman harga tiga paket (Free/Pro/Enterprise) + toggle bulanan/tahunan (diskon 15%).
- Fitur change request: pencarian, filter, dan ekspor CSV pada daftar domain.

## Batasan yang Diketahui

- Pembayaran paket Pro/Enterprise belum terintegrasi payment gateway; pemesanan
  dilakukan melalui WhatsApp dan aktivasi oleh tim.
- Infrastruktur mitigasi DDoS skala produksi (multi-Tbps) tidak dibangun penuh;
  prototipe berfokus pada alur inti, aturan bisnis, dan dasbor.
- Integrasi otomatis dengan penyedia DNS/CDN pihak ketiga belum ada.
- SLA 99,9% dan deployment on-premise masih sebatas penawaran paket Enterprise.

## Struktur Folder

| File / folder | Isi |
|---|---|
| `docs/` | Dokumen proses kerja (spesifikasi, rencana, log prompt, kasus uji, dan lainnya) |
| `docs/bukti/` | Tangkapan layar bukti pengujian alur inti |
| `dashboard/` | Kode website & dashboard (Next.js + React + TypeScript) |
| `internal/`, `cmd/` | Engine proteksi GarudaShield (Go) |
| `tes/` | Unit test website |
| `bughunt-kasir/` | Hasil Bug Hunt (kode, test, `catatan-bug.md`) |

## Link Demo (opsional)

- Demo publik: https://gg.garudashield.my.id (jika tersedia)

## Sumber dan Lisensi

| Nama | Sumber (link) | Lisensi |
|---|---|---|
| Next.js | https://nextjs.org | MIT |
| React | https://react.dev | MIT |
| Tailwind CSS | https://tailwindcss.com | MIT |
| Motion | https://motion.dev | MIT |
| Go | https://go.dev | BSD-3-Clause |
| Redis | https://redis.io | BSD-3-Clause (RSALv2/SSPL untuk edisi baru) |

## Tim

| Nama | Peran |
|---|---|
| Adriansyah | Backend & keamanan (registrasi/login, aturan paket, unit test) |
| Muhammad Didan Apdila | Frontend & UI (portal/dashboard, validasi domain, dokumentasi) |
| Muhammad Fahish Alfayyadh | Full-stack & integrasi (halaman harga, pantau trafik, change request) |
