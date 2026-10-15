# Kesalahan AI yang Kami Temukan

Catatan ini bukan daftar kegagalan tim, melainkan bukti bahwa kami memeriksa hasil AI
sebelum menerimanya.

| No | Apa yang diminta | Kesalahan AI | Bagaimana kami mengetahuinya | Perbaikan yang kami lakukan |
|---|---|---|---|---|
| 1 | Aturan diskon (member vs. non-member) | AI menggabungkan diskon member 10% dan diskon belanja 5% sehingga total menjadi 15% | Uji dengan keranjang besar + status member memberi diskon 15%, padahal spesifikasi: hanya satu diskon berlaku | Ubah logika agar member memakai 10% saja, non-member memakai 5% saja |
| 2 | Batas diskon non-member "minimal Rp100.000" | AI memakai `subtotal > 100000`, sehingga tepat Rp100.000 tidak dapat diskon | Dibandingkan dengan aturan "minimal Rp100.000" (termasuk nilai sama) | Ubah menjadi `subtotal >= 100000` |
| 3 | Pembulatan PPN 11% | AI memakai `Math.floor` sehingga PPN dibulatkan ke bawah | Contoh: dasar Rp11.550 → PPN 1.270,5; AI memberi 1.270, seharusnya 1.271 | Ubah menjadi pembulatan ke rupiah terdekat (`Math.round`) |
| 4 | Validasi jumlah pesanan | AI tidak memvalidasi jumlah; nilai kosong/0/negatif/pecahan diterima | Memasukkan jumlah kosong dan negatif lewat form, tidak ada pesan kesalahan | Tambahkan validasi bilangan bulat 1–99 dengan pesan kesalahan |
| 5 | Data domain setelah muat ulang | AI menyimpan daftar domain hanya di state memori komponen | Muat ulang halaman → daftar domain hilang | Pindahkan penyimpanan agar persisten |
| 6 | Panel pantau trafik real-time | AI memakai interval polling 500 ms sehingga CPU/browser berat | Halaman terasa berat dan permintaan menumpuk di tab Network | Jadikan interval 2–5 detik dan hentikan polling saat tab tidak aktif |
| 7 | Header keamanan halaman | AI memasang `style-src 'self'` pada CSP sehingga font Google terblokir | Halaman kehilangan font dan muncul error CSP di konsol | Izinkan `fonts.googleapis.com` (style) & `fonts.gstatic.com` (font) |
| 8 | Ekspor CSV | AI tidak memberi kutip pada nilai yang mengandung koma sehingga kolom bergeser | Membuka CSV di spreadsheet, jumlah kolom tidak sesuai | Bungkus nilai ber-koma dengan tanda kutip dan uji ulang |
| 9 | Anggota tim duplikat | AI memetakan anggota duplikat ke status **400**, bukan **409** | `go test ./internal/admin/...` gagal: tes lama berharap 409 (duplikat = konflik) | Petakan error unik Postgres (`23505`) ke **409 Conflict** (konsisten dgn aset/onboard) |
| 10 | Portal setelah muat ulang | AI hanya memakai `useState`; sesi tidak dipulihkan saat mount sehingga pengguna dengan cookie valid kembali ke form login | Muat ulang `/portal` → kembali ke form "Masuk Portal" walau sudah login | Tambah `useEffect` saat mount: panggil `tenant/me`; valid → buka portal, tidak valid → tetap di kartu auth tanpa error |
| 11 | Dokumentasi hasil uji | AI sempat menandai kolom "Hasil" gagal hanya karena label "Paket Aktif" ter-`uppercase` oleh CSS saat dicek | Membandingkan teks DOM vs label yang tampil; data aktual benar (FREE) | Periksa nilai sebenarnya (respons backend + isi DOM) sebelum menyimpulkan |

**Pelajaran:** pada beberapa kasus AI cepat menghasilkan struktur, tetapi aturan
bisnis (pembulatan, batas "minimal", diskon tunggal) dan kasus tepi (input salah)
harus selalu dihitung manual oleh tim lalu diuji.
