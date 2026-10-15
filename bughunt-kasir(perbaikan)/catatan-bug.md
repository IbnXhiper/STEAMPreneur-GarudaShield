# Catatan Bug Hunt: Kasir Kantin STEAM

**Nama tim: Garuda Shield**

Dokumen ini mencatat bug yang benar-benar ditemukan, diuji, dan diperbaiki pada aplikasi Kasir Kantin STEAM. Seluruh perilaku mengacu pada aturan bisnis resmi di `README.md`. Test ditulis dengan modul bawaan Node.js (`node:test`, `node:assert/strict`) dan dijalankan dengan `node --test` dari folder `bughunt-kasir`.

## Ringkasan hasil test

| Kode | tests | pass | fail |
|---|---|---|---|
| Kode asli (sebelum perbaikan) | 42 | 18 | 24 |
| Kode setelah perbaikan | 42 | 42 | 0 |

## Daftar bug

| No | Judul | Aturan dilanggar | Berkas |
|---|---|---|---|
| 1 | Jumlah pesanan tidak divalidasi (qty berupa string) | 1 | `kasir.js` |
| 2 | Batas maksimum 99 per produk tidak ditegakkan saat penggabungan | 2 | `kasir.js` |
| 3 | Diskon member dan diskon non-member digabung | 4 | `kasir.js` |
| 4 | Ambang diskon non-member memakai `>` bukan `>=` | 4 | `kasir.js` |
| 5 | PPN dibulatkan ke bawah (`Math.floor`) bukan ke terdekat | 5 | `kasir.js` |
| 6 | Pembayaran kurang tidak ditolak | 7 | `kasir.js` |

---

## Bug 1 — Jumlah pesanan tidak divalidasi

- **Gejala** (apa yang terlihat salah):
  Jumlah yang tidak sah tetap diterima. Nilai `0`, negatif, pecahan, `NaN`, dan `Infinity` semuanya masuk ke keranjang. Karena `app.js` mengirim nilai dari `<input type="number">` sebagai string, jumlah produk disimpan sebagai string (misalnya `"2"`), sehingga penjumlahan berikutnya menjadi penggabungan string (`"2" + "3" = "23"`).

- **Langkah untuk memunculkan bug:**
  1. Tambahkan Es Teh (`K04`) dengan jumlah `2`.
  2. Tambahkan Es Teh lagi dengan jumlah `3`.
  3. Jumlah di keranjang tertulis `"23"`, bukan `5`.
     - Setara dalam kode: `tambahItem(k, "K04", "2")` lalu `tambahItem(k, "K04", "3")` → `k[0].qty === "23"`.
  4. Tambahkan produk dengan jumlah `0`, `-1`, `1.5`, `NaN`, `Infinity`, `""`, atau `"   "` → semuanya diterima tanpa error.

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 1 — Jumlah pesanan per produk harus bilangan bulat dari 1 sampai 99. Jumlah kosong, nol, negatif, atau pecahan harus ditolak dengan pesan kesalahan. (Batas atas 99 untuk item tunggal juga termasuk di sini.)

- **Penyebab** (file, fungsi, dan penjelasan):
  Fungsi `tambahItem` pada `kasir.js` (kode lama, baris 25–43) langsung memakai nilai `qty` tanpa memeriksa tipe maupun rentangnya. Nilai disimpan apa adanya (`qty: qty`), dan pada penggabungan memakai operator `+=`. Ketika `qty` berupa string (sebagaimana dikirim `app.js`), `itemLama.qty += qty` melakukan konkatenasi string, sehingga `2 + "3"` menghasilkan `"23"`, bukan `5`. Tidak ada pemeriksaan `Number.isInteger` atau batas 1–99, sehingga nol, negatif, pecahan, `NaN`, dan `Infinity` lolos ke keranjang. Ini melanggar aturan 1 karena jumlah yang tidak sah seharusnya ditolak di titik masuk.

- **Perbaikan** (apa yang diubah):
  Pada `kasir.js` (baris 31–34), nilai `qty` dikonversi dengan `Number(qty)` lalu divalidasi:
  ```js
  const jumlah = Number(qty);
  if (!Number.isInteger(jumlah) || jumlah < 1 || jumlah > 99) {
    throw new Error("Jumlah harus bilangan bulat antara 1 dan 99");
  }
  ```
  Nilai yang disimpan kemudian adalah `jumlah` (angka), bukan `qty` mentah. Konversi eksplisit ini memperbaiki perilaku konkatenasi string sekaligus menolak nilai tidak sah. `Number("")`, `Number(" ")`, dan `Number("abc")` menghasilkan `0`/`NaN` yang otomatis ditolak oleh rentang 1–99, sehingga input kosong tidak diam-diam dianggap nol yang sah.

- **Test yang membuktikan** (nama test):
  Berkas `tes/validasi-keranjang.test.js`: "jumlah 0 ditolak", "jumlah negatif ditolak", "jumlah pecahan ditolak", "jumlah NaN ditolak", "jumlah Infinity ditolak", "jumlah string kosong ditolak (tidak diam-diam jadi 0)", "jumlah string spasi ditolak (tidak diam-diam jadi 0)", "jumlah string 'abc' ditolak", "jumlah 100 ditolak (di atas batas 99)", "jumlah berupa string angka '2' diterima dan dikonversi ke angka", dan "tambah produk sama dari input string menjumlahkan qty sebagai angka (2 + '3' = 5)".
  Hasil pada kode lama: **gagal** (contoh: `'2' !== 2` dan `'23' !== 5`; test penolakan tidak melempar error).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: validasi jumlah pesanan 1-99 dan konversi tipe angka`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Bug 2 — Batas maksimum 99 per produk tidak ditegakkan saat penggabungan

- **Gejala** (apa yang terlihat salah):
  Jumlah satu jenis produk bisa melebihi 99 ketika produk yang sama ditambahkan berulang kali. Contoh: tambah 98 lalu tambah 2 → jumlah menjadi `100`, padahal seharusnya ditolak.

- **Langkah untuk memunculkan bug:**
  1. Tambahkan Es Teh 98.
  2. Tambahkan Es Teh 2.
  3. Jumlah di keranjang menjadi `100` (melampaui batas 99). Seharusnya operasi ditolak dan keranjang tetap 98.

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 2 — Jika produk yang sudah ada di keranjang ditambahkan lagi, jumlahnya dijumlahkan, dan total jumlah per produk tetap tidak boleh lebih dari 99.

- **Penyebab** (file, fungsi, dan penjelasan):
  Pada kode lama di `kasir.js`, cabang penggabungan di fungsi `tambahItem` (baris 31–34) hanya menjalankan `itemLama.qty += qty`. Tidak ada pemeriksaan apakah hasil penjumlahan melewati 99, sehingga 98 + 2 = 100 diterima. Pemeriksaan batas hanya dilakukan pada item baru (lewat rentang) tidak ada sama sekali, sehingga aturan 2 soal batas gabungan tidak pernah ditegakkan.

- **Perbaikan** (apa yang diubah):
  Pada `kasir.js` (baris 36–41), sebelum mengubah keranjang, hasil gabungan diperiksa lebih dulu:
  ```js
  if (itemLama.qty + jumlah > 99) {
    throw new Error("Jumlah maksimal 99 per produk");
  }
  itemLama.qty += jumlah;
  ```
  Pemeriksaan dilakukan **sebelum** mutasi sehingga operasi yang ditolak tidak meninggalkan keranjang dalam keadaan berubah sebagian.

- **Test yang membuktikan** (nama test):
  Berkas `tes/validasi-keranjang.test.js`: "98 + 1 = 99 diterima", "98 + 2 melebihi batas dan ditolak", "99 + 1 melebihi batas dan ditolak", "operasi merge yang ditolak tidak mengubah keranjang (tanpa perubahan sebagian)", dan "tambah 3 x 99 tetap terkunci di 99 dan gagal pada penambahan setelah penuh".
  Hasil pada kode lama: **gagal** (jumlah menjadi 100; test penolakan tidak melempar error).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: tegakkan batas maksimum 99 saat menggabungkan produk`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Bug 3 — Diskon member dan diskon non-member digabung

- **Gejala** (apa yang terlihat salah):
  Member dengan subtotal minimal Rp100.000 mendapat diskon **15%** (5% + 10%), padahal seharusnya hanya 10%. Contoh: subtotal 151.000 → diskon tertulis 22.650, seharusnya 15.100.

- **Langkah untuk memunculkan bug:**
  1. Centang "Pelanggan member".
  2. Isi keranjang hingga subtotal Rp151.000.
  3. Diskon yang dihitung = 22.650 (15%). Seharusnya 15.100 (10%).

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 4 — Diskon tidak boleh digabung, hanya satu yang berlaku. Member mendapat diskon 10% dari subtotal; non-member dengan subtotal minimal Rp100.000 mendapat diskon 5%.

- **Penyebab** (file, fungsi, dan penjelasan):
  Pada kode lama, fungsi `hitungDiskon` di `kasir.js` (baris 58–67) memiliki dua blok `if` terpisah yang dijalankan berurutan. Blok pertama menambahkan diskon 5% bila `subtotal > 100000`, blok kedua menambahkan diskon 10% bila `member`. Karena keduanya menambahkan ke variabel `diskon` yang sama, pelanggan member dengan subtotal tinggi mendapat 5% + 10% = 15%. Ini melanggar aturan 4 karena diskon seharusnya saling eksklusif, bukan dijumlahkan.

- **Perbaikan** (apa yang diubah):
  Pada `kasir.js` (baris 66–74), logika diubah menjadi `if / else if` agar hanya satu diskon yang berlaku:
  ```js
  if (member) {
    diskon = (subtotal * 10) / 100;
  } else if (subtotal >= 100000) {
    diskon = (subtotal * 5) / 100;
  }
  return Math.round(diskon);
  ```

- **Test yang membuktikan** (nama test):
  Berkas `tes/diskon.test.js`: "member di atas 100000 hanya dapat 10%, tidak digabung dengan 5%" dan "member tepat 100000 hanya dapat 10% (10000), tidak digabung 15%".
  Hasil pada kode lama: **gagal** (`22650 !== 15100`).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: diskon member dan non-member tidak boleh digabung`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Bug 4 — Ambang diskon non-member memakai `>` bukan `>=`

- **Gejala** (apa yang terlihat salah):
  Non-member dengan subtotal **tepat** Rp100.000 tidak mendapat diskon, padahal aturan menyebut "minimal Rp100.000" berarti diskon berlaku mulai dari (termasuk) Rp100.000.

- **Langkah untuk memunculkan bug:**
  1. Pastikan "Pelanggan member" tidak dicentang.
  2. Isi keranjang hingga subtotal tepat Rp100.000 (misalnya Es Teh 25 × Rp4.000).
  3. Diskon = 0. Seharusnya 5% × 100.000 = Rp5.000.

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 4 — Pelanggan bukan member dengan subtotal minimal Rp100.000 mendapat diskon 5% dari subtotal.

- **Penyebab** (file, fungsi, dan penjelasan):
  Pada kode lama, fungsi `hitungDiskon` di `kasir.js` memakai perbandingan ketat `subtotal > 100000` (baris 60). Karena ketat, subtotal yang tepat 100.000 tidak memenuhi syarat, sehingga diskon dihitung 0. Kata "minimal Rp100.000" pada aturan 4 berarti batas harus inklusif, sehingga memakai `>` mengecualikan nilai batas dan melanggar aturan tersebut.

- **Perbaikan** (apa yang diubah):
  Perbandingan diubah menjadi `subtotal >= 100000` pada `kasir.js` (baris 70), sehingga batas Rp100.000 inklusif sesuai kata "minimal".

- **Test yang membuktikan** (nama test):
  Berkas `tes/diskon.test.js`: "non-member tepat 100000 dapat diskon 5% (5000)" dan (regresi) "non-member di bawah 100000 tidak dapat diskon".
  Berkas `tes/ppn-total.test.js`: "ringkasan non-member tepat 100000 (diskon 5000, PPN 10450, total 105450)".
  Hasil pada kode lama: **gagal** (`0 !== 5000`).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: ambang diskon non-member minimal Rp100.000 inklusif`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Bug 5 — PPN dibulatkan ke bawah, bukan ke rupiah terdekat

- **Gejala** (apa yang terlihat salah):
  Nilai PPN selalu dibulatkan ke bawah, sehingga pecahan 0,5 ke atas tidak dibulatkan naik. Contoh: dasar pajak Rp950 → PPN = 104,5 → tertulis 104, seharusnya 105. Dasar Rp960 → 105,6 → tertulis 105, seharusnya 106.

- **Langkah untuk memunculkan bug:**
  1. Panggil `hitungPPN(950)` → hasil 104. Seharusnya 105 (0,5 dibulatkan naik).
  2. Panggil `hitungPPN(960)` → hasil 105. Seharusnya 106 (di atas 0,5 dibulatkan naik).

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 5 — PPN 11% dihitung dari (subtotal dikurangi diskon), lalu dibulatkan ke rupiah terdekat dengan pecahan 0,5 ke atas dibulatkan naik.

- **Penyebab** (file, fungsi, dan penjelasan):
  Pada kode lama, fungsi `hitungPPN` di `kasir.js` (baris 70–72) memakai `Math.floor((dasar * 11) / 100)`. `Math.floor` selalu membulatkan ke bawah, jadi pecahan berapa pun—termasuk yang ≥ 0,5—dibuang. Akibatnya PPN tidak pernah dibulatkan ke rupiah terdekat sebagaimana diminta aturan 5, sehingga nilai pajak terlalu kecil untuk dasar pajak yang menghasilkan pecahan ≥ 0,5.

- **Perbaikan** (apa yang diubah):
  Pada `kasir.js` (baris 78) ditambahkan `+ 0.5` sebelum `Math.floor`:
  ```js
  return Math.floor((dasar * 11) / 100 + 0.5);
  ```
  Untuk bilangan positif, menambah 0,5 lalu membulatkan ke bawah menghasilkan pembulatan setengah ke atas. Untuk pembilang bilangan bulat, pecahan tepat 0,5 (misalnya 104,5 dari 950 × 11 / 100) direpresentasikan tepat sebagai bilangan biner sehingga hasilnya deterministik (105).

- **Test yang membuktikan** (nama test):
  Berkas `tes/ppn-total.test.js`: "PPN pecahan di bawah 0,5 dibulatkan turun (940 -> 103,4 -> 103)", "PPN pecahan tepat 0,5 dibulatkan naik (950 -> 104,5 -> 105)", "PPN pecahan di atas 0,5 dibulatkan naik (960 -> 105,6 -> 106)", dan "PPN bilangan bulat tetap sama (1000 -> 110)".
  Hasil pada kode lama: **gagal** (`104 !== 105` dan `105 !== 106`).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: PPN dibulatkan ke rupiah terdekat (0,5 ke atas)`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Bug 6 — Pembayaran kurang tidak ditolak

- **Gejala** (apa yang terlihat salah):
  Ketika uang yang dibayar kurang dari total, transaksi justru dianggap berhasil dan menampilkan kembalian negatif. Contoh: total 10.500, dibayar 10.499 → aplikasi menampilkan "Kembalian: Rp-1" dan pesan "Pembayaran berhasil".

- **Langkah untuk memunculkan bug:**
  1. Isi keranjang dengan total Rp10.500.
  2. Isi "Uang dibayar" dengan 10.499.
  3. Klik "Bayar" → muncul "Kembalian: Rp-1" dan pesan "Pembayaran berhasil". Seharusnya ditolak dengan pesan "Uang pembayaran kurang".

- **Aturan bisnis yang dilanggar** (nomor aturan di README):
  Aturan 7 — Jika uang yang dibayar kurang dari total, transaksi ditolak dengan pesan "Uang pembayaran kurang". Jika cukup, kembalian = uang dibayar - total.

- **Penyebab** (file, fungsi, dan penjelasan):
  Pada kode lama, fungsi `hitungKembalian` di `kasir.js` (baris 83–85) hanya mengembalikan `bayar - total` tanpa pemeriksaan. Di `app.js`, hasil dari `hitungKembalian` dianggap sukses selama tidak ada error, lalu ditampilkan "Kembalian: ..." dan pesan "Pembayaran berhasil". Karena tidak ada validasi, pembayaran kurang menghasilkan kembalian negatif dan tetap dianggap berhasil, melanggar aturan 7 yang mengharuskan transaksi ditolak.

- **Perbaikan** (apa yang diubah):
  Pada `kasir.js` (baris 90–95) ditambahkan pemeriksaan sebelum menghitung kembalian:
  ```js
  if (bayar < total) {
    throw new Error("Uang pembayaran kurang");
  }
  return bayar - total;
  ```
  Karena `app.js` menangkap error dan menampilkan `err.message`, pesan yang tampil persis "Uang pembayaran kurang" dan teks kembalian dibersihkan (`elHasilBayar.textContent = ""`), sehingga transaksi yang ditolak tidak menampilkan status sukses maupun kembalian yang menyesatkan.

- **Test yang membuktikan** (nama test):
  Berkas `tes/pembayaran.test.js`: "pembayaran kurang Rp1 ditolak dengan pesan persis", "pembayaran jauh di bawah total ditolak", "pembayaran nol ditolak (uang dibayar kosong -> 0)", "pembayaran tepat sama menghasilkan kembalian 0", "pembayaran lebih menghasilkan kembalian yang benar", dan "transaksi yang ditolak tidak menghasilkan kembalian menyesatkan".
  Hasil pada kode lama: **gagal** (tidak ada exception yang dilempar).
  Hasil setelah perbaikan: **lulus**.

- **Commit perbaikan** (pesan commit):
  Belum dibuat — menunggu commit manual.
  Usulan: `perbaikan: tolak pembayaran kurang dengan pesan "Uang pembayaran kurang"`

- **Ditemukan oleh:** bersama (AI dibantu pemeriksaan manusia)

---

## Test tambahan yang ditambahkan

| Berkas | Cakupan |
|---|---|
| `tes/validasi-keranjang.test.js` | Validasi jumlah (aturan 1) dan penggabungan/batas 99 (aturan 2) |
| `tes/diskon.test.js` | Diskon member & non-member, ambang 100000, pembulatan (aturan 4) |
| `tes/ppn-total.test.js` | Pembulatan PPN (aturan 5) dan total (aturan 6) |
| `tes/pembayaran.test.js` | Pembayaran kurang/tepat/lebih dan kembalian (aturan 7) |

Test contoh `tes/contoh.test.js` tidak diubah dan tetap lulus.

## Berkas yang diubah

- `kasir.js` — perbaikan 6 bug di atas (fungsi `tambahItem`, `hitungDiskon`, `hitungPPN`, `hitungKembalian`).
- `catatan-bug.md` — dokumen ini.
- `tes/validasi-keranjang.test.js`, `tes/diskon.test.js`, `tes/ppn-total.test.js`, `tes/pembayaran.test.js` — test baru.

`app.js` dan `index.html` tidak diubah karena perilaku tampilan sudah benar setelah logika di `kasir.js` diperbaiki.
