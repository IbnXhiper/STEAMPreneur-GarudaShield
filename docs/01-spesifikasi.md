# **Spesifikasi Produk**

Nama tim: GarudaShield
Nama produk: GarudaShield
Alur inti yang dibangun: Pelanggan memilih paket produk, lalu mengkonfigurasi domain website nya dan system garuda shield akan melindungi website pelanggan dari ancaman serangan DDoS dan ancaman lainnya, sekaligus menyediakan dasbor untuk memantau lalu lintas situs web.
## 1. Masalah
Masalah apa yang diselesaikan? Siapa yang mengalaminya?
Website dapat terganggu akibat lonjakan trafik atau serangan DDoS yang membebani server hingga layanan sulit diakses. Kondisi ini menyulitkan pemilik website dan tim IT dalam menjaga ketersediaan layanan.
## 2. Pengguna
Siapa yang memakai aplikasi ini? Apa yang mereka butuhkan?
GarudaShield ditujukan bagi pemilik website, pelaku UMKM, administrator server, dan tim IT. Mereka membutuhkan sistem untuk memantau trafik, mendeteksi aktivitas mencurigakan, serta melakukan mitigasi terhadap potensi serangan DDoS.

## 3. Hasil wawancara calon pengguna
Narasumber (tulis perannya saja, tanpa data pribadi, misalnya "pelanggan kantin, kelas XI"):
- Network Engineer — instansi pemerintah daerah bidang komunikasi dan informatika
- Programmer — instansi pemerintah daerah bidang komunikasi dan informatika
- Data Analyst — instansi pemerintah daerah bidang komunikasi dan informatika
- Fullstack Developer — praktisi industri
- Praktisi Cyber Security — praktisi keamanan siber

Cara wawancara (langsung / lewat pesan) dan tanggal:
- 5 Oktober 2026, pukul 14.00 WIB — Tatap muka: tiga narasumber dari instansi pemerintah (Network Engineer, Programmer, dan Data Analyst) diwawancarai dalam satu sesi.
- 5 Oktober 2026, pukul 20.00 WIB — Tatap muka: wawancara bersama narasumber dengan peran Fullstack Developer.
- 6 Oktober 2026, pukul 20.00 WIB — Daring (Google Meet): wawancara bersama narasumber dengan peran Praktisi Cyber Security.

| Pertanyaan kami | Jawaban narasumber |
| :--- | :--- |
|Kalau boleh tahu, selama bekerja di bidang IT peran Bapak saat ini di bidang IT itu sebagai apa ya?|Narasumber memiliki peran sebagai Network Engineer, Programmer, Data Analyst, Fullstack Developer, dan Praktisi Cyber Security.|
|Pernah mengalami gangguan karena banyaknya permintaan yang masuk secara bersamaan?|Tiga narasumber dari instansi pemerintah pernah mengalaminya saat membangun layanan yang dipakai masyarakat; Praktisi Cyber Security menyebut kondisi traffic tinggi sudah menjadi "makanan sehari-hari" dalam pekerjaannya.|
|Kalau terjadi kondisi seperti itu, biasanya apa yang paling dikhawatirkan oleh tim IT?|Server kewalahan atau down sehingga layanan tidak dapat diakses, serta risiko kehilangan atau terganggunya data.|
|Kalau kondisi seperti itu terjadi, biasanya perusahaan melakukan apa untuk mengatasinya?|Instansi menggunakan firewall; Fullstack Developer menekankan pentingnya keamanan berlapis (multi-layer); Praktisi Cyber Security menjelaskan mitigasi pada Layer 4 (jaringan/transport) dan Layer 7 (aplikasi).|
|Menurut Bapak/Ibu, apakah permasalahan yang ingin kami selesaikan ini memang relevan dengan kebutuhan perusahaan?|Semua narasumber menilai masalah tersebut relevan dengan kebutuhan keamanan dan ketersediaan layanan digital.|
|Apakah GarudaShield cocok untuk membantu perusahaan menghadapi masalah di atas?|GarudaShield dinilai berpotensi membantu, tetapi perlu melalui pengujian teknis terlebih dahulu dan dipastikan tidak ada kesalahan. Penerapannya juga perlu mempertimbangkan fitur, harga, kesesuaian kebutuhan, dan kemampuan menghadapi serangan DDoS berskala besar.|

**Catatan hasil wawancara:** Para narasumber memvalidasi masalahnya (status: **tervalidasi**), namun menegaskan bahwa efektivitas GarudaShield tidak cukup dinilai dari daftar fitur — perlu pembuktian lewat pengujian teknis dan evaluasi biaya. Masukan ini menjadi dasar aturan bisnis dan kriteria "selesai" pada bagian berikutnya.

## 4. Fitur wajib (hanya untuk alur inti)
## Alur Inti

GarudaShield menggunakan model layanan berlangganan (*subscription-based*) dengan tiga pilihan paket, yaitu Free, Pro, dan Enterprise. Setiap pengguna yang mendaftar secara otomatis mendapatkan paket **Free** sebagai akses awal untuk mencoba fitur proteksi dasar GarudaShield.

1. **Registrasi dan Login** — Pengguna membuat akun dan masuk ke dashboard GarudaShield. Setelah registrasi berhasil, sistem secara otomatis memberikan paket Free kepada pengguna.
2. **Menambahkan Domain** — Pengguna mendaftarkan domain website yang ingin dilindungi dan melakukan konfigurasi sesuai petunjuk yang diberikan GarudaShield.
3. **Aktivasi Proteksi** — Pengguna menyelesaikan proses verifikasi dan konfigurasi domain agar layanan proteksi dapat diaktifkan sesuai kemampuan paket yang digunakan.
4. **Pemantauan Keamanan** — Pengguna memantau status proteksi, lalu lintas website, dan aktivitas mencurigakan melalui dashboard secara real-time sesuai fitur paketnya.
5. **Deteksi dan Mitigasi Ancaman** — GarudaShield menganalisis lalu lintas masuk untuk mengidentifikasi potensi serangan DDoS, bot, dan permintaan mencurigakan, kemudian menerapkan mekanisme mitigasi sesuai fitur paket aktif.
6. **Upgrade Paket** — Pengguna dapat meningkatkan paket Free ke Pro untuk memperoleh proteksi yang lebih lengkap, termasuk mitigasi DDoS L4 dan L7, reputasi IP dan challenge, GeoIP global berbasis negara dan ASN, serta proteksi anti-bot dan crawler.
7. **Pengelolaan Langganan** — Pengguna mengelola paket dan masa berlangganan melalui dashboard. Untuk kebutuhan skala besar, pengguna dapat menghubungi tim GarudaShield untuk paket Enterprise yang menyediakan opsi multi-node, SLA, dan deployment on-premise.

**Target bisnis:** Paket Free berfungsi sebagai pintu masuk agar pengguna dapat mencoba layanan GarudaShield, sedangkan paket Pro menjadi target konversi utama melalui penawaran fitur keamanan yang lebih lengkap untuk UMKM dan bisnis online di Indonesia.

## 5. Di luar cakupan

- Implementasi infrastruktur mitigasi DDoS skala produksi yang mampu menangani serangan lalu lintas nyata secara masif.
- Integrasi langsung dengan penyedia DNS, CDN, atau layanan keamanan pihak ketiga untuk mengaktifkan proteksi domain secara otomatis.
- Sistem pembayaran dan penagihan berlangganan secara nyata untuk paket Pro dan Enterprise.
- Implementasi infrastruktur terdistribusi (multi-node), deployment on-premise, dan jaminan SLA 99,9%.
- Pengembangan seluruh mekanisme keamanan tingkat lanjut secara lengkap, termasuk analisis perilaku berbasis AI dan sistem deteksi ancaman adaptif untuk lingkungan produksi.
- Pengujian ketahanan terhadap serangan DDoS berskala besar menggunakan lalu lintas serangan nyata.

## 6. Aturan bisnis dan perhitungan

- Setiap pengguna yang berhasil mendaftar secara otomatis mendapatkan paket Free tanpa biaya.
- Paket Free memiliki batas maksimal 1 domain website untuk setiap akun.
- Paket Free menyediakan fitur WAF dan proteksi DDoS L7 dasar, GeoIP 1 negara, dashboard real-time, rate limit standar, dan dukungan komunitas.
- Pengguna yang membutuhkan fitur lebih lengkap dapat meningkatkan paket ke Pro dengan harga Rp499.000 per bulan atau memilih paket tahunan dengan diskon 15%.
- Paket Pro mencakup seluruh fitur Free, mitigasi DDoS L4 dan L7, reputasi IP dan challenge, GeoIP global berbasis negara dan ASN, proteksi anti-bot dan crawler, serta dukungan email 24/7.
- Paket Enterprise ditujukan untuk kebutuhan khusus dengan fitur seperti sinkronisasi multi-node, TLS termination dan custom domain, SLA 99,9%, serta deployment on-premise atau dedicated sesuai kesepakatan.
- Status paket pengguna hanya berubah menjadi Pro setelah pembayaran terkonfirmasi. Jika pembayaran belum terkonfirmasi, paket pengguna tetap mengikuti status langganan sebelumnya.
- Penambahan domain harus melalui validasi. Domain yang kosong atau tidak sesuai format ditolak dengan pesan kesalahan yang jelas.
- Pengguna hanya dapat mengakses fitur yang tersedia pada paket aktifnya. Fitur berbayar yang tidak termasuk dalam paket Free harus menampilkan informasi upgrade.
- Status proteksi domain ditampilkan berdasarkan status konfigurasi dan aktivasi yang tercatat dalam sistem. Status berhasil tidak boleh ditampilkan sebelum persyaratan aktivasi terpenuhi.

## 7. Data yang disimpan
| **Data** | **Contoh isi** | **Disimpan di mana** |
| :--- | :--- | :--- |
| Data akun pengguna | ID pengguna, nama, email | Database atau localStorage untuk prototipe |
|informasi paket|Nama paket: Free, Pro, atau Enterprise|Database atau localStorage|
|Status langganan|Gratis, menunggu pembayaran, aktif, atau kedaluwarsa|Database atau localStorage|
|Data domain|ID domain, nama domain, pemilik domain|Database atau localStorage|
|Status verifikasi domain|Belum diverifikasi, berhasil, atau gagal|Database atau localStorage|
|Status proteksi|Belum aktif, aktif, atau mengalami gangguan|Database atau localStorage|
|Konfigurasi proteksi|Waktu penambahan domain dan perubahan paket|Database atau localStorage|
|Data transaksi|ID transaksi, nominal pembayaran, status pembayaran|Database atau localStorage|

- ## 8. Kriteria "selesai"

Setiap kriteria harus dapat diuji dan dijawab ya atau tidak.

- [x] Pengguna dapat mendaftar dan login, kemudian mendapatkan paket Free secara otomatis setelah registrasi berhasil.
- [x] Pengguna dapat menambahkan domain dengan input yang valid, sedangkan input kosong atau format domain yang tidak valid ditolak dengan pesan kesalahan yang jelas.
- [x] Sistem membatasi pengguna paket Free maksimal satu domain dan menampilkan informasi upgrade ketika pengguna mencoba menambahkan domain berikutnya.
- [x] Data akun, paket, dan domain tersimpan serta tetap tersedia setelah halaman dimuat ulang.
- [x] Dashboard menampilkan daftar domain beserta status verifikasi dan proteksinya sesuai data yang tersimpan.
- [x] Sistem menerapkan aturan akses fitur berdasarkan paket pengguna dan menampilkan informasi upgrade untuk fitur yang tidak tersedia pada paket Free.
- [x] Pengguna dapat melihat informasi paket Pro dengan harga Rp499.000 per bulan dan opsi paket tahunan dengan diskon 15%.
- [x] Seluruh kasus uji yang direncanakan telah dijalankan, hasil aktual dicatat, dan setiap kriteria yang dinyatakan lulus memiliki bukti pengujian (lihat `04-kasus-uji.md`).
- [x] Website dapat dijalankan mengikuti instruksi di README.md tanpa langkah yang terlewat.