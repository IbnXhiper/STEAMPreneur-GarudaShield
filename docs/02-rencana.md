# Rencana Kerja

Kami membangun GarudaShield secara bertahap: mulai dari satu alur yang benar-benar
berfungsi (registrasi → tambah domain → tampil di dashboard), lalu menambah fitur
sedikit demi sedikit. Setiap langkah dijaga kecil agar bisa dikerjakan dan diperiksa
dalam 30–60 menit.

**Waktu mulai:** 1 Oktober 2026
**Batas akhir pengumpulan:** 12 Oktober 2026

| No | Langkah | Penanggung jawab | Cara memeriksa bahwa langkah ini selesai | Estimasi waktu | Waktu sebenarnya | Status |
|---|---|---|---|---|---|---|
| 1 | Scaffolding: buat struktur proyek (`dashboard/`, `docs/`, `tes/`), set konstanta paket Free/Pro/Enterprise | Didan | Proyek bisa dibuka di browser di http://localhost:3000 | 45 menit | 50 menit | Selesai |
| 2 | Wawancara calon pengguna & validasi masalah | Fahish | Catatan wawancara terisi di `01-spesifikasi.md`, status masalah tervalidasi | 90 menit | 2 sesi (5–6 Okt) | Selesai |
| 3 | Halaman harga (Free/Pro/Enterprise) + toggle bulanan/tahunan (diskon 15%) | Didan | Harga Pro tampil Rp499.000/bln; tahunan tampil potongan 15% | 60 menit | 55 menit | Selesai |
| 4 | Registrasi & login; akun baru otomatis dapat paket Free | Adriansyah | Daftar akun baru → cek paket otomatis "Free"; salah password ditolak | 60 menit | 75 menit | Selesai |
| 5 | Tambah domain + validasi format (kosong/format salah ditolak) | Adriansyah | Domain valid tersimpan; input kosong/salah tampil pesan kesalahan | 60 menit | 60 menit | Selesai |
| 6 | Aturan batas paket Free (maks 1 domain) + info upgrade | Adriansyah | Domain ke-2 pada akun Free ditolak dengan tawaran upgrade | 45 menit | 40 menit | Selesai |
| 7 | Dashboard: daftar domain + status verifikasi & proteksi, tersimpan setelah reload | Didan | Muat ulang halaman → data domain tetap ada | 60 menit | 60 menit | Selesai |
| 8 | Panel pantau trafik & ancaman (polling/SSE) | Fahish | Angka trafik berubah mengikuti aktivitas | 60 menit | 80 menit | Selesai |
| 9 | Unit test alur inti (registrasi, batas domain, validasi) | Adriansyah | `go test ./internal/... ./cmd/...` hijau | 60 menit | 55 menit | Selesai |
| 10 | Bug Hunt | Seluruh tim | Catatan bug dan test selesai (`catatan-bug.md`) | 60 menit | | Belum |
| 11 | Change request (pencarian, filter, ekspor CSV) | Adriansyah | Semua kasus uji lama dan baru lulus | 90 menit | 75 menit | Selesai |
| 12 | Dokumentasi dan refleksi | Fahish | README bisa diikuti orang lain | 60 menit | 50 Menit | Selesai |

**Catatan waktu:** Langkah yang paling melewati estimasi adalah langkah 8 (panel
pantau trafik) karena menyesuaikan data real-time dengan tampilan, dan langkah 4
(registrasi/login) karena menangani kasus salah password dan sesi.