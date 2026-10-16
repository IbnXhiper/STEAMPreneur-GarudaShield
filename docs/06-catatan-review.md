# Catatan Code Review

Minta AI me-review kode kalian, lalu putuskan saran mana yang diterima. Menolak
saran dengan alasan jelas juga hal yang baik.

**Prompt review yang dipakai:**
"Review kode untuk pendaftaran pengguna, validasi domain, dan aturan paket Free.
Sebutkan risiko keamanan, duplikasi, dan kasus tepi yang belum ditangani. Beri
contoh konkret dan jangan ubah kode tanpa persetujuan."

| No | Saran dari AI | Diterima / ditolak | Alasan | Bagian kode yang berubah |
|---|---|---|---|---|
| 1 | Validasi panjang & kekuatan password saat registrasi | Diterima | Mencegah akun lemah; mudah diuji | fungsi validasi registrasi |
| 2 | Tolak domain duplikat untuk akun yang sama | Diterima | Sesuai aturan bisnis, mencegah data ganda | fungsi tambah domain |
| 3 | Gunakan rate limit login untuk mencegah percobaan password berulang | Diterima | Melindungi dari brute force; tetap ringan | handler login |
| 4 | Simpan daftar domain secara persisten, bukan hanya di state | Diterima | Memenuhi kriteria "data tetap ada setelah muat ulang" | penyimpanan domain |
| 5 | Asingkan (refactor) seluruh aturan paket ke satu modul | Ditolak | Untuk skala challenge ini, memindahkan semuanya menambah risiko regresi tanpa manfaat nyata; kami cukup memusatkan konstanta paket | tidak ada perubahan |
| 6 | Tambahkan library state management pihak ketiga | Ditolak | Berlebihan untuk kebutuhan alur inti; menambah dependensi dan ukuran berkas | tidak ada perubahan |
| 7 | Ganti validasi format domain dengan library validasi eksternal penuh | Ditolak (sementara) | Validasi sederhana kami sudah memenuhi aturan; library menambah bobot dan belum tentu akurat untuk kebutuhan ini | tidak ada perubahan |
| 8 | Hapus penanganan fallback saat penyimpanan eksternal tidak tersedia | Ditolak | Fallback justru penting: juri dapat menjalankan tanpa Redis/PostgreSQL | tidak ada perubahan |

**Ringkasan:** 4 saran diterima, 4 ditolak dengan alasan. Penjelasan kode lengkap ada
di `08-penjelasan-kode.md`.
