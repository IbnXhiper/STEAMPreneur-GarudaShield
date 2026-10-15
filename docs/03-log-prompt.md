# Log Prompt

**Alat AI yang dipakai:** Claude (opencode) sebagai asisten utama, ChatGPT untuk
bahasa/dokumen, dan GitHub Copilot untuk melengkapi potongan kode.

Kami hanya mencatat prompt penting: prompt yang menghasilkan kode utama, prompt yang
hasilnya ditolak/diperbaiki, dan prompt untuk debugging.

Catatan ini kini dibatasi hanya pada **Change Request**: pencarian (search), filter,
dan ekspor CSV pada daftar domain. Entri sebelum pengerjaan CR (pembuatan proyek,
spesifikasi, rencana, kasus uji, dan fitur sebelumnya) serta entri di luar CR (bug
hunt `kasir.js`, integrasi form registrasi) sudah dihapus.

| No | Waktu | Langkah ke- | Prompt (ringkas) | Hasil dari AI | Keputusan tim (diterima / diubah / ditolak) dan alasannya |
|---|---|---|---|---|---|
| 1 | 14 Okt 2026 | CR | "Tambahkan pencarian, filter status, dan ekspor CSV pada tabel domain." | Fungsi filter + tombol ekspor CSV | **Diterima** setelah memperbaiki format CSV (koma pada data diberi tanda kutip) |

**Rincian prompt penuh** disimpan pada riwayat sesi tim; tabel di atas memuat yang
paling berpengaruh pada hasil akhir.
