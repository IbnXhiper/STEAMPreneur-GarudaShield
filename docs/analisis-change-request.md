# Analisis Change Request

Isi bagian ini **sebelum** mengubah kode, lalu commit dan push (contoh pesan:
`dokumen: analisis change request`). Juri memeriksa bahwa commit ini muncul sebelum
commit perubahan kode.

**Change request:** pencarian, filter, dan ekspor CSV (lihat Tugas 3 di lembar
challenge).

**Kriteria penerimaan yang akan dipenuhi dan cara kami memenuhinya:**

1. Pengguna dapat **mencari** domain berdasarkan nama domain (kata kunci). → tambah
   kotak pencarian di atas tabel daftar domain; pencarian tidak membedakan huruf
   besar/kecil.
2. Pengguna dapat **memfilter** domain berdasarkan status (verifikasi/proteksi). →
   tambah pilihan filter status; daftar hanya menampilkan baris yang cocok.
3. Pengguna dapat **mengekspor CSV** dari daftar yang **sedang tampil** (setelah
   pencarian/filter). → tombol "Ekspor CSV" yang membangkitkan berkas dari data
   terfilter dan mengunduh berkasnya.
4. Pencarian atau filter yang tidak menemukan apa pun menampilkan pesan "tidak ada
   domain" (bukan tabel kosong tanpa keterangan).

## 1. Dampak

| Bagian yang terdampak | File / fungsi | Perubahan yang diperlukan |
|---|---|---|
| Spesifikasi | `docs/01-spesifikasi.md` (bagian alur inti & kriteria) | Tambah langkah pencarian/filter/ekspor pada pengelolaan domain |
| Kode (tampilan) | `dashboard/app/portal/page.tsx` (komponen daftar domain) | Tambah state kata kunci + status filter, terapkan sebelum render, dan tombol ekspor |
| Kode (data) | helper daftar domain (fungsi filter & pembuat CSV) | Tambah fungsi `filterDomains` dan `toCSV` |
| Kasus uji / test | `docs/04-kasus-uji.md`, `tes/` | Tambah U10 + unit test untuk filter dan format CSV |
| README | `README.md` (bagian Fitur) | Cantumkan fitur pencarian/filter/ekspor CSV |

## 2. Kasus uji baru

| ID | Langkah / input | Hasil yang diharapkan |
|---|---|---|
| U10a | Ketik kata kunci `toko` pada kotak cari | Hanya domain yang namanya memuat "toko" yang tampil |
| U10b | Pilih filter status "belum diverifikasi" | Hanya domain berstatus belum diverifikasi yang tampil |
| U10c | Gabungkan kata kunci + filter yang tidak cocok | Muncul pesan "tidak ada domain", tabel kosong |
| U10d | Klik "Ekspor CSV" setelah filter aktif | Berkas CSV berisi persis baris yang tampil; nilai ber-koma diberi tanda kutip |
| U10e | Cari/filter lalu bersihkan (reset) | Seluruh domain tampil kembali |
| U10f | Klik "Ekspor CSV" saat daftar kosong | Muncul pemberitahuan bahwa tidak ada data untuk diekspor |

## 3. Setelah selesai

- [x] Kasus uji lama tetap lulus
- [x] Kasus uji baru lulus
- [x] Spesifikasi diperbarui
- [x] README diperbarui
