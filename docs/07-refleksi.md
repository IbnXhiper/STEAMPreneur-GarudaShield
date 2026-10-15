# Refleksi Tim

1. **Apa yang berjalan baik dalam cara kami bekerja dengan AI?**
   AI sangat membantu untuk membuat kerangka cepat: struktur halaman, komponen
   dashboard, dan penulisan test boilerplate. Kami juga terbantu saat debugging
   karena AI bisa menjelaskan penyebab error. Yang paling berhasil adalah kebiasaan
   menuliskan aturan bisnis di spesifikasi lebih dulu, lalu memakai aturan itu untuk
   memeriksa hasil AI — bukan sekadar menerima kode yang "jalan".

2. **Kapan AI paling membantu? Kapan AI justru menyulitkan?**
   Paling membantu saat pekerjaan berulang dan jelas (membuat form, tabel, format
   CSV, menulis contoh test). Justru menyulitkan saat aturan bisnis halus: AI
   menggabungkan diskon, memakai pemotongan (floor) alih-alih pembulatan, dan
   melewatkan kasus tepi seperti input kosong. Pada momen itu kami harus menghitung
   manual dengan kalkulator.

3. **Kesalahan terbesar yang hampir lolos, dan bagaimana kami menemukannya?**
   Dua hal: (a) diskon member dan non-member yang digabung menjadi 15%, dan (b)
   PPN yang memakai `Math.floor`. Keduanya hanya ketahuan ketika kami menguji kasus
   perhitungan di `04-kasus-uji.md` secara manual dan mencocokkan dengan aturan
   spesifikasi, bukan hanya menjalankan aplikasi.

4. **Apa yang akan kami lakukan berbeda jika mengulang challenge ini?**
   Menulis kasus uji (Bagian A) lebih lengkap sebelum menulis kode, terutama kasus
   batas ("minimal Rp100.000", PPN tepat 0,5) dan input salah. Kami juga akan
   menetapkan satu orang sebagai "penjaga aturan bisnis" yang memeriksa setiap
   perubahan terhadap spesifikasi.

5. **Bagaimana kami mengatur waktu? Langkah mana yang jauh lebih lama dari
   estimasi, dan kenapa?**
   Kami bekerja dalam beberapa sesi, memakai rencana per langkah 30–60 menit
   (`02-rencana.md`). Langkah yang melewati estimasi: panel pantau trafik real-time
   (langkah 8) dan registrasi/login (langkah 4), karena keduanya banyak kasus tepi
   (polling yang membebani, salah password, sesi). Menggabungkan aturan paket dan
   validasi juga memakan waktu karena harus diuji ulang setelah setiap perubahan.

6. **Bagian mana dari kode yang masih belum kami pahami sepenuhnya?**
   Bagian penyimpanan persisten/sinkronisasi antar-permukaan (state store) masih
   paling sulit kami pahami — khususnya bagaimana data tetap konsisten ketika
   penyimpanan eksternal mati. Kami memahaminya dengan membaca ulang alur dan
   menambah log, tetapi detail internalnya belum sepenuhnya kami kuasai.
