# Penjelasan Kode

Dokumen ini adalah bukti utama bahwa tim memahami kode yang dihasilkan AI. Setiap
anggota menjelaskan minimal dua bagian kode yang berbeda, memakai bahasa sendiri,
dengan nama file dan fungsi yang nyata serta contoh nilai nyata.

---

## Bagian A: Penjelasan per anggota

### Penjelasan 1
- **Ditulis oleh:** Adriansyah
- **File dan fungsi:** `dashboard/app/pricing/page.tsx` — komponen `PricingPage`,
  array `tiers`, dan state `billing`.
- **Apa yang dilakukan:** Menampilkan tiga paket (Free, Pro, Enterprise) dari array
  `tiers`. Setiap item menyimpan `id`, `name`, `price`, `tagline`, dan daftar
  `features`. State `billing` bernilai `"monthly"` atau `"yearly"` dan mengubah harga
  yang tampil.
- **Kenapa dibuat seperti ini:** Data paket dipisahkan ke satu array agar halaman dan
  aturan bisnis memakai sumber yang sama; tidak perlu mengubah JSX bila harga/fitur
  berubah.
- **Contoh:** bila `billing === "monthly"`, paket Pro menampilkan "Rp 499.000/bln".
  Bila diubah ke `"yearly"`, untuk `t.id === "pro"` harga menjadi "Rp 5.089.000/thn"
  (Rp499.000 × 12 = Rp5.988.000, dikurangi diskon 15% ≈ Rp898.200).
- **Jika input salah atau kosong:** array `tiers` bersifat tetap sehingga selalu
  terisi; jika `billing` bernilai tak dikenal, hanya kondisi `"yearly" && pro` yang
  spesifik yang berubah, selebihnya kembali ke `t.price` (nilai aman).

### Penjelasan 2
- **Ditulis oleh:** Adriansyah
- **File dan fungsi:** `internal/auth/account.go` — `(*Handler).Register`.
- **Apa yang dilakukan:** Menerima data pendaftaran (email, nama, password),
  memvalidasi, lalu membuat tenant baru dengan `plan = 'free'`, membuat user pemilik
  dengan password yang di-hash, dan mencatat langganan. Mengembalikan JSON sukses
  berisi `{"ok": true, "tenant": ..., "plan": "free"}`.
- **Kenapa dibuat seperti ini:** Aturan bisnis menyatakan setiap akun baru otomatis
  mendapat paket Free, jadi nilai `'free'` ditulis langsung saat insert dan bukan
  diketik pengguna. Pengecekan keunikan email memakai deteksi pelanggaran unique
  (`isUniqueViolation`) agar tidak terjadi akun ganda.
- **Contoh:** registrasi `demo@garudashield.test` dengan password valid → tenant
  dibuat, respons `plan` = `"free"`.
- **Jika input salah atau kosong:** password terlalu pendek atau email tidak valid
  ditolak dengan status `400` dan pesan kesalahan; email yang sudah terdaftar
  ditolak sebagai duplikat. Tidak ada akun yang dibuat sebagian (operasi dibatalkan).

### Penjelasan 3
- **Ditulis oleh:** Muhammad Didan Apdila
- **File dan fungsi:** `internal/policy/policy.go` — `(*Policy).Validate`.
- **Apa yang dilakukan:** Memeriksa nilai konfigurasi proteksi per domain sebelum
  disimpan: rentang `Mode`, `BudgetMs`, `RateLimit`, `Geo`, dan sebagainya. Bila ada
  nilai di luar rentang atau tidak dikenal, mengembalikan `error`.
- **Kenapa dibuat seperti ini:** Pola *fail-closed*: konfigurasi yang tidak valid
  harus ditolak sejak awal, bukan diterima lalu menimbulkan perilaku proteksi yang
  lemah. Validasi dipisah dari penyimpanan agar mudah diuji.
- **Contoh:** `BudgetMs` bernilai `-5` (di bawah nol) ditolak karena nol/negatif
  berarti fail-closed instan dan harus disengaja; nilai wajar seperti `25` diterima.
- **Jika input salah atau kosong:** nilai kosong/`0` pada field opsional memakai
  default global (tidak mengubah proteksi global); nilai yang secara eksplisit salah
  (mis. mode tak dikenal) menghasilkan error dan policy tidak disimpan.

### Penjelasan 4
- **Ditulis oleh:** Muhammad Didan Apdila
- **File dan fungsi:** `dashboard/app/api/gs/[...path]/route.ts` — handler BFF
  `GET`, `POST`, dan `DELETE`.
- **Apa yang dilakukan:** Meneruskan permintaan dari browser ke engine GarudaShield
  (`/api/gs/<x>` → `/api/<x>`), menyuntikkan kunci admin **di sisi server**, dan
  mengembalikan respons. Mendukung streaming (SSE) dan membuang header hop-by-hop.
- **Kenapa dibuat seperti ini:** Kunci admin tidak boleh ada di kode browser
  (client); BFF menjadi satu-satunya jalur aman. Parameter `[...path]` (catch-all)
  membuat satu berkas melayani banyak endpoint.
- **Contoh:** permintaan dashboard `gsGet("stats")` menjadi `GET /api/gs/stats` →
  BFF menambahkan `X-GS-Key` lalu memanggil engine `GET /api/stats` → JSON diteruskan.
- **Jika input salah atau kosong:** `path` kosong/tak dikenal diteruskan ke engine
  yang mengembalikan `404`/`400`; BFF mengembalikan status tersebut apa adanya
  sehingga dashboard dapat menampilkan pesan yang tepat.

### Penjelasan 5
- **Ditulis oleh:** Muhammad Fahish Alfayyadh
- **File dan fungsi:** `internal/shield/evaluate.go` — `(*Shield).Evaluate`.
- **Apa yang dilakukan:** Orkestrator utama alur evaluasi setiap permintaan:
  whitelist → blocklist → token `gs_ok` → emergency → L4 → L7 → WAF → anomali →
  keputusan. Setiap tahap dapat menghasilkan `Decision` (pass/challenge/block) sesuai
  skor dan aturan.
- **Kenapa dibuat seperti ini:** Urutan tahap menentukan paritas dan keamanan
  (fail-closed). Evaluasi dibatasi anggaran waktu (`BudgetMs`); bila anggaran habis
  pada konteks tertentu, keputusan diarahkan ke sisi aman.
- **Contoh:** permintaan dengan pola SQLi terdeteksi WAF menambah skor ≥50 → verdict
  `block`; permintaan bersih dengan skor <20 → `pass` diteruskan ke origin.
- **Jika input salah atau kosong:** permintaan tanpa IP valid atau dengan header
  rusak tidak lolos tahap awal; `gs_ok` yang tidak valid diabaikan (bukan diterima),
  dan permintaan lanjut dievaluasi normal alih-alih diberi fast-path.

### Penjelasan 6
- **Ditulis oleh:** Muhammad Fahish Alfayyadh
- **File dan fungsi:** `internal/shield/domains.go` — `(*Shield).SetDomainPolicy`
  dan `(*Shield).runtimeFor`.
- **Apa yang dilakukan:** `SetDomainPolicy` menyimpan/memperbarui policy per domain
  pada registry runtime; `runtimeFor` mengambil runtime (`rt`) yang tepat untuk sebuah
  `host`. Runtime dihitung dengan menyalin konfigurasi global lalu menimpa (overlay)
  policy domain.
- **Kenapa dibuat seperti ini:** Setiap domain dapat punya knob berbeda, tetapi
  pengaturan global harus tetap berlaku bila policy domain kosong (fail-closed). Peta
  runtime di memori mempercepat hot path.
- **Contoh:** domain `toko-umkm.id` dengan policy `Mode: emergency` → `runtimeFor`
  mengembalikan `rt` yang emergency; permintaan berikutnya ke host itu dievaluasi
  dalam mode darurat tanpa mengubah perilaku domain lain.
- **Jika input salah atau kosong:** `host` kosong/tidak terdaftar memakai runtime
  global (tanpa overlay); policy yang gagal divalidasi tidak pernah masuk registry
  sehingga tidak melemahkan proteksi.

---

## Bagian B: Pertanyaan yang dijawab bersama

1. **Jika aturan bisnis utama berubah — misalnya harga Pro naik dan diskon tahunan
   menjadi 20% — file/fungsi mana yang harus diubah?**
   Harga dan diskon berada di `dashboard/app/pricing/page.tsx` (array `tiers` dan
   logika `billing`). Cukup ubah harga pada item Pro dan perhitungan diskon; aturan
   batas domain Free berada pada jalur tambah domain. Karena nilai dipusatkan, tidak
   perlu menyentuh banyak berkas.

2. **Bagaimana data mengalir dari form sampai tampil di layar?**
   Pengguna mengisi form (mis. tambah domain) → handler di komponen React memvalidasi
   ringan → permintaan dikirim lewat `gsPost` (`dashboard/lib/api.ts`) → BFF
   `dashboard/app/api/gs/[...path]/route.ts` menyuntikkan kunci admin → engine
   memproses (`internal/policy`/`internal/shield`) → respons JSON kembali melalui BFF
   → state diperbarui → komponen merender ulang tabel.

3. **Pilih satu unit test. Apa yang diuji, dan kapan test itu gagal?**
   Test `internal/auth/account_test.go` untuk `Register` menguji bahwa akun baru
   mendapat plan `free`. Test ini akan gagal jika insert tidak menetapkan `'free'`,
   validasi password dilonggarkan, atau respons tidak memuat `plan: free`.

4. **Satu saran AI yang kami tolak, dan kenapa?**
   AI menyarankan memindahkan seluruh aturan paket ke library state management pihak
   ketiga. Kami menolaknya karena untuk skala challenge ini hal itu menambah
   dependensi dan ukuran berkas tanpa manfaat nyata, serta berisiko regresi.

5. **Bagian kode yang paling sulit dipahami, dan bagaimana akhirnya dipahami?**
   Bagian penyimpanan dua lapis (memori + penyimpanan eksternal) di `internal/
   statestore`: bagaimana data tetap konsisten ketika penyimpanan eksternal tidak
   tersedia. Kami memahaminya dengan membaca alur `Evaluate` → `runtimeFor`, menambah
   log, dan mencoba mematikan penyimpanan eksternal untuk melihat perilaku fallback.
