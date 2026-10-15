# Kasus Uji

Bagian A diisi **sebelum** mulai membangun. Hasil yang diharapkan dihitung sendiri
oleh tim (boleh dengan kalkulator), bukan oleh AI. Bagian B diisi setelah aplikasi
diuji.

| ID | Kriteria yang diuji | Langkah / input | Hasil yang diharapkan | Hasil sebenarnya | Lulus? | Bukti (nama file tangkapan layar) |
|---|---|---|---|---|---|---|
| U01 | Registrasi otomatis paket Free | Daftar akun baru dengan email & password valid, lalu login | Akun dibuat dan paket yang tampil = **Free** | Akun dibuat, paket "Free" tampil di portal | Ya | `bukti/u01-registrasi-free.png` |
| U02 | Login menolak password salah | Login dengan password salah | Login ditolak, muncul pesan kesalahan, tidak masuk dashboard | Ditolak dengan pesan kesalahan | Ya | `bukti/u02-login-salah.png` |
| U03 | Tambah domain valid | Tambah domain `addxpkg.jscorp.web.id` pada akun Free | Domain tersimpan, tampil di daftar dengan status verifikasi "belum diverifikasi" | Tersimpan & tampil | Ya | `bukti/u03-tambah-domain.png` |
| U04 | **Input salah:** domain kosong | Tambah domain dengan kolom nama domain dibiarkan kosong | Ditolak, muncul pesan "Domain wajib diisi" | Ditolak dengan pesan jelas | Ya | `bukti/u04-domain-kosong.png` |
| U05 | **Input salah:** format domain salah | Tambah domain `fahish com` (ada spasi, tanpa titik) | Ditolak, muncul pesan format domain tidak valid | Ditolak dengan pesan jelas | Ya | `bukti/u05-domain-salah.png` |
| U06 | **Di batas aturan:** Free maks 1 domain | Akun Free sudah punya 1 domain, lalu tambah domain ke-2 `domain-anda.com` | Ditolak, muncul informasi upgrade ke Pro | Ditolak + tawaran upgrade | Ya | `bukti/u06-batas-free.png` |
| U07 | Harga Pro & diskon tahunan | Buka halaman harga; aktifkan toggle "Tahunan" | Bulanan: **Rp499.000/bln**; Tahunan: potongan **15%** (Rp499.000 × 12 = Rp5.988.000; diskon 15% = Rp898.200; ≈ **Rp5.089.000/thn**) | Angka sesuai perhitungan | Ya | `bukti/u07-harga.png` |
| U08 | Data tetap ada setelah muat ulang | Tambah 1 domain, lalu muat ulang (F5) halaman portal | Domain tetap tampil setelah reload | Domain tetap tampil | Ya | `bukti/u08-reload.png` |
| U09 | Change request: cari/filter/ekspor | Isi kata kunci "asset" di kotak cari; filter status "belum diverifikasi"; klik "Ekspor CSV" | Daftar tersaring sesuai kata kunci & status; file CSV berisi baris domain yang tampil | Daftar tersaring & CSV benar (kolom tidak bergeser) | Ya | `bukti/u10-change-request.png` |

Wajib ada:
- minimal satu kasus dengan **input yang salah** (kosong, negatif, huruf di kolom
  angka) → **U04, U05**;
- minimal satu kasus **di batas aturan** (contoh: tepat Rp50.000 jika aturannya
  "minimal Rp50.000") → **U06** (tepat pada batas 1 domain paket Free) dan **U07**
  (perhitungan diskon).

## Unit test otomatis (jika ada)

- Nama file test: `internal/auth/account_test.go`, `internal/policy/policy_test.go`,
  `internal/shield/domains_test.go`, dan `tes/domain.test.js`
- Jumlah test: 38 lulus dari 38 (0 gagal)

Cara menjalankan:
```
go test ./internal/... ./cmd/... -count=1
cd dashboard && npm run build
```
