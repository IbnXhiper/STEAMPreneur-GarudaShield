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