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
