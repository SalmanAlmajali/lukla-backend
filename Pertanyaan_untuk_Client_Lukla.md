# Pertanyaan Klarifikasi untuk Lukla Manufacturing × Snøhund

**Proyek:** Lukla Operations Engine
**Tanggal:** 12 Oktober 2026
**Dasar:** SRS Walkthrough v2.0 (12 Okt 2026)

---

## Tentang dokumen ini

Kami sudah membaca SRS dengan teliti. Sebagian besar sudah jelas. Namun ada bagian yang **belum cukup rinci atau saling berbeda**, sehingga kami belum bisa menggambar alur kerja (diagram proses) yang benar dan menyusun kebutuhan sistem tanpa menebak.

Dokumen ini berisi pertanyaan-pertanyaan tersebut. Tujuannya agar sistem yang dibangun **sesuai cara kerja pabrik Anda yang sebenarnya**, bukan menurut asumsi kami.

### Cara menjawab
- Tulis jawaban di bawah setiap pertanyaan (di baris **Jawaban:**). Jawaban singkat saja sudah cukup.
- Jika ada **pilihan**, centang satu atau tulis jawaban sendiri.
- Jika belum tahu atau belum diputuskan, tulis **"Belum diputuskan"** dan sebutkan siapa yang akan memutuskan serta kira-kira kapan.
- **Contoh dokumen nyata sangat membantu** (foto buku catatan, Excel, form kertas, slip gaji, nota). Lihat Bagian 12.

### Tingkat prioritas
| Kode | Arti | Harapan |
|---|---|---|
| **P1** | Menentukan cara kerja inti sistem. Kami tidak bisa membangun bagian terkait tanpa jawabannya. | Dijawab lebih dulu |
| **P2** | Penting untuk kelengkapan, tetapi bisa kami lanjutkan dengan asumsi sementara. | Dijawab sebelum modul terkait dikerjakan |
| **P3** | Bisa dibahas belakangan atau di tahap berikutnya. | Boleh menyusul |

---

## Ringkasan: 10 pertanyaan paling mendesak (semua P1)

Jika waktu terbatas, mohon jawab ini lebih dulu:

1. **Q4.1**: Siapa yang menginput hasil kerja: mandor atau penjahit sendiri?
    - mandor.
2. **Q4.3**: Bagaimana line (kelompok penjahit) dibentuk dan apakah penjahit bisa pindah line?
    - tidak ada aturan pengelompokan penjahit, yang penting penjahit untuk order client ada dengan maksimal sekitar 70 mesin.
    - penjahit bisa pindah line tentunya dengan kondisi yang memerlukan dia pindah, seperti mesin rusak dan lainnya.
3. **Q5.1**: Istilah dan alur hasil QC yang baku (lolos, repair, defect/reject/scrap).
    - lolos: barang lolos QC
    - repair: barang perlu diperbaiki
    - reject: barang ditolak dan tidak bisa diperbaiki
    - defect: berisi kategori defect untuk barang yang tidak lolos
4. **Q5.4**: Apakah barang yang ditolak QC mempengaruhi upah penjahit?
    - tidak mempengaruhi upah 
5. **Q6.1**: "Rate card" itu upah penjahit, tarif ke klien, atau keduanya?
    - untuk upah penjahit dan tarif ke klien
6. **Q6.4**: Siapa itu "Kepala Operasional" dan siapa dua Mandor penerima bonus QC?
    - kepala operasional yang mengawasi berjalan operasional.
    - dua mandor dan kepala operasional yang dapat bonus QC adalah yang memantau line produksi. 
7. **Q6.5**: Berapa nilai bonus per pcs dan bagaimana cara menghitungnya?
    - berdasarkan rate bonus yang diisi oleh owner
    - rate bonus dikali jumlah barang lolos QC
8. **Q6.6**: Uang makan 10–15% itu dibagikan ke siapa dan dengan cara apa?
    - untuk penjahit
9. **Q8.1**: Apa yang terjadi setelah barang lolos QC (pengiriman ke klien)?
    - memberikan notifikasi barang telah selesai ke client, minimal seperti memberikan update status di halaman client.
10. **Q12.1**: Contoh dokumen/form yang dipakai saat ini.
    - tidak ada contoh dokumen, boleh diasumsikan kolom yang mungkin terlibat tetapi perlu diberikan penjelasan pemilihan kolom tersebut. 
---

## 1. Gambaran Umum & Pengambil Keputusan

**Q1.1 — Siapa pengambil keputusan?** *(P1)*
Sistem ini dimiliki bersama oleh tim Operasional Lukla dan Manajemen Brand Snøhund. Agar keputusan cepat, kami perlu satu nama untuk tiap area.
- Pengambil keputusan untuk urusan **operasional pabrik** (produksi, gudang, QC, payroll): …
- Pengambil keputusan untuk urusan **brand/produk Snøhund**: …
- Nama **Business Sponsor / pemilik** proyek: …

**Jawaban:**

**Q1.2 — Siapa penanggung jawab teknis dari pihak Lukla?** *(P2)*
Siapa yang bisa kami hubungi untuk pertanyaan sehari-hari dan sesi pengujian (misalnya wakil dari Gudang, Mandor, QC, dan Finance)?

**Jawaban:**

**Q1.3 — Target waktu** *(P2)*
Apakah ada tanggal target mulai pakai (go-live) atau batas waktu tertentu (misalnya sebelum musim produksi tertentu)?

**Jawaban:**

---

## 2. Sample & Persetujuan Order

*Rujukan SRS: Fase 1 dan 2 (Sample Phase, Approval & Mass Order).*

**Q2.1 — Siapa yang menyetujui sample?** *(P1)*
Di SRS, Tim Sample boleh "menyetujui versi sample". Apakah persetujuan itu cukup dari Tim Sample, atau klien/brand juga harus menyetujui sebelum produksi massal?
- [ ] Cukup Tim Sample
- [ ] Perlu persetujuan klien/brand (bentuknya: foto, tanda tangan, pesan WhatsApp, dsb.): …
- [ ] Lainnya: …

**Jawaban:**

**Q2.2 — Siapa yang menyetujui order produksi massal?** *(P1)*
SRS menyebut Tim Sample **tidak boleh** menyetujui pesanan massal "tanpa konfirmasi". Siapa yang berwenang memberi konfirmasi itu (Owner, Kepala Produksi, atau lainnya)?

**Jawaban:**

**Q2.3 — Apa yang disimpan pada setiap versi sample?** *(P2)*
Untuk "Sample v1, v2, dst." kami perkirakan perlu disimpan: spesifikasi, pemakaian bahan, waktu pengerjaan per komponen, dan catatan perubahan. Apakah ada yang kurang atau berlebih? Apakah **foto/gambar/lampiran** perlu disimpan juga?

**Jawaban:**

**Q2.4 — Time study** *(P2)*
Bagaimana time study dilakukan saat ini? (Siapa yang mengukur, berapa kali per komponen, dan hasil akhirnya dipakai untuk apa: menentukan tarif jahit, estimasi waktu, atau lainnya?)

**Jawaban:**

**Q2.5 — Perubahan setelah produksi massal dimulai** *(P2)*
Jika klien meminta perubahan spesifikasi atau jumlah setelah produksi massal berjalan, apa yang biasanya dilakukan? Apakah order bisa dibatalkan atau diubah?

**Jawaban:**

---

## 3. Pembelian & Penerimaan Bahan (Gudang)

*Rujukan SRS: Fase 3 (Gudang & Cutting) dan aturan anti-kontaminasi silang.*

**Q3.1 — Pembelian bahan oleh Lukla** *(P1)*
Untuk order **Semi-CMT** dan **Full Package**, Lukla menyediakan sebagian atau seluruh bahan. SRS menyebut Finance dapat "membuat/menyetujui PO bahan", tetapi tidak ada penjelasan alurnya. Mohon jelaskan:
- Siapa yang menentukan kebutuhan bahan (berdasarkan apa)?
- Siapa yang membuat dan menyetujui PO ke pemasok?
- Bagaimana bahan yang dibeli dicatat masuk gudang dan dikaitkan ke order tertentu?
- Apakah pembuatan PO **harus ada di sistem** pada rilis pertama, atau cukup dicatat di luar sistem dahulu?

**Jawaban:**

**Q3.2 — Penerimaan bahan titipan** *(P2)*
Apa yang dicek dan dicatat saat bahan klien datang (jumlah, kondisi, satuan: meter/kg/roll/pcs)? Apakah ada surat jalan/bukti terima yang diberikan ke klien? Bagaimana jika jumlah yang datang tidak sesuai?

**Jawaban:**

**Q3.3 — Stok opname** *(P2)*
Seberapa sering stok opname dilakukan (harian, mingguan, bulanan)? Jika hasil hitung berbeda dengan catatan sistem, siapa yang menyetujui penyesuaian stok?

**Jawaban:**

**Q3.4 — Sisa potongan (scrap/waste)** *(P2)*
- Apa saja **kategori** sisa potongan yang ingin dicatat (misalnya sisa potong, kain cacat, sisa perca)?
- Apakah sisa bahan dikembalikan ke klien, disimpan, dijual, atau dibuang?

**Jawaban:**

**Q3.5 — Bahan yang tertukar atau salah keluar** *(P2)*
Saat sistem menolak pengeluaran bahan karena pemilik tidak cocok, apa yang harus dilakukan petugas gudang selanjutnya? Apakah kejadian itu perlu dilaporkan ke seseorang (misalnya Kepala Produksi atau Owner)?

**Jawaban:**

---

## 4. Produksi, Line & Pencatatan Output 30 Menit

*Rujukan SRS: Fase 4 (Line Produksi), pencatatan output 30 menit, dan peran Penjahit/Mandor.*

**Q4.1 — Siapa yang menginput hasil kerja?** *(P1)*
SRS menyebut dua hal yang berbeda: tampilan input 30 menit dibuat untuk **mandor**, tetapi izin juga menyebut **penjahit** dapat "melaporkan hasil kerja (diverifikasi mandor)". Cara mana yang benar?
- [ ] **A.** Mandor selalu yang menginput hasil semua penjahit.
- [ ] **B.** Penjahit melapor sendiri (lewat tablet/HP), lalu mandor memverifikasi.
- [ ] **C.** Campuran: …

Jika B atau C: apakah penjahit punya HP/tablet sendiri, atau akan memakai tablet bersama di tiap line?

**Jawaban:**

**Q4.2 — Kapan verifikasi dilakukan?** *(P1)*
Upah dihitung dari output yang "diverifikasi mandor". Kapan verifikasi dilakukan: setiap kali input 30 menit, akhir hari, atau sebelum gaji Sabtu? Bagaimana jika ada angka yang salah, siapa yang boleh mengoreksi dan sampai kapan?

**Jawaban:**

**Q4.3 — Apa itu "line" dan bagaimana dibentuk?** *(P1)*
SRS menyebut "line produksi paralel" tanpa penjelasan. Mohon jelaskan:
- Satu line berisi berapa penjahit, dan bagaimana dibentuk (per order, per artikel, per keahlian)?
- Apakah satu line hanya mengerjakan **satu order/artikel** pada satu waktu?
- Apakah satu line punya **satu mandor**?
- Apakah penjahit **boleh pindah line** di tengah hari/minggu? Jika ya, bagaimana output dan upahnya dicatat?
- Apakah ada pembagian tahap dalam satu line (misalnya penjahit lengan, kerah, saku, perakitan)?

**Jawaban:**

**Q4.4 — Pembagian kerja per komponen** *(P1)*
Satu produk jadi (misalnya satu kemeja) biasanya dikerjakan oleh beberapa penjahit yang masing-masing memegang satu komponen. Apakah benar demikian? Siapa yang menentukan penjahit mana mengerjakan komponen apa?

**Jawaban:**

**Q4.5 — Daftar artikel dan komponen** *(P1)*
Kami perlu daftar **artikel/produk** dan **komponen** tiap artikel (misalnya Kemeja A: lengan, kerah, badan depan, badan belakang, saku). Apakah sudah ada daftarnya? Siapa yang akan menambah artikel baru di kemudian hari?

**Jawaban:**

**Q4.6 — Jam kerja** *(P2)*
Berapa jam kerja dan shift per hari? Apakah ada lembur? Apakah input 30 menit berlaku sepanjang jam kerja, termasuk saat istirahat?

**Jawaban:**

**Q4.7 — Penjahit baru dan penjahit keluar** *(P1)*
Skenario uji SRS menambah 5 penjahit di tengah minggu. Siapa yang mendaftarkan penjahit baru ke sistem, dan data apa yang dicatat (nama, nomor, tanggal masuk, rate khusus, dan sebagainya)? Bagaimana jika penjahit berhenti di tengah minggu?

**Jawaban:**

---

## 5. Quality Control (QC)

*Rujukan SRS: Fase 5 (QC & Barang Jadi).*

**Q5.1 — Istilah status hasil QC yang baku** *(P1)*
Di SRS, istilahnya berbeda-beda di beberapa tempat:
- Alur operasional: **Lolos / Repair / Defect**
- Hak akses QC: **lolos / reject / repair**
- Tampilan layar QC: **lolos / repair / scrap**

Mohon tentukan istilah yang dipakai dan artinya masing-masing:

| Istilah | Artinya (menurut Lukla) | Apa yang terjadi pada barangnya? |
|---|---|---|
| Lolos | | |
| Repair | | |
| Defect / Reject / Scrap | | |

**Jawaban:**

**Q5.2 — Alur barang repair** *(P1)*
Barang yang di-repair dikerjakan oleh siapa (penjahit asal atau tim khusus)? Setelah selesai, apakah diperiksa ulang oleh QC? Apakah ada batas berapa kali boleh repair?

**Jawaban:**

**Q5.3 — Barang defect/reject** *(P1)*
Apa yang dilakukan pada barang yang tidak bisa diperbaiki (dibuang, dijual murah, dikembalikan ke klien, dihitung sebagai kerugian siapa)? Siapa yang menanggung biayanya?

**Jawaban:**

**Q5.4 — Dampak QC terhadap upah penjahit** *(P1)*
Jika barang hasil jahitan seorang penjahit ditolak atau harus di-repair, apakah upahnya:
- [ ] Tetap dibayar penuh
- [ ] Dipotong sebagian/seluruhnya
- [ ] Dibayar penuh tetapi dihitung sebagai catatan kualitas
- [ ] Lainnya: …

**Jawaban:**

**Q5.5 — Penelusuran ke sumber** *(P1)*
Saat QC menemukan cacat, apakah QC perlu tahu **penjahit/line/batch** mana yang mengerjakannya? Jika ya, sampai tingkat mana yang diinginkan?
- [ ] Penjahit (nama)
- [ ] Line
- [ ] Batch (kelompok hasil kerja per 30 menit)
- [ ] Tidak perlu
- [ ] Lainnya: …

Karena satu produk bisa dikerjakan beberapa penjahit, apakah QC perlu mencatat **komponen mana** yang cacat (misalnya lengan, kerah)?

**Jawaban:**

**Q5.6 — Cara pemeriksaan** *(P2)*
Apakah QC memeriksa **setiap produk** atau hanya sampel? Apakah QC dilakukan di akhir proses atau juga di tengah jalan (inline)? Apakah satu produk bisa diperiksa lebih dari sekali?

**Jawaban:**

**Q5.7 — Daftar jenis cacat** *(P2)*
Apa saja jenis cacat yang ingin dicatat (misalnya jahitan lepas, ukuran salah, noda, lubang)? Apakah sudah ada daftarnya?

**Jawaban:**

---

## 6. Payroll, Bonus & Biaya

*Rujukan SRS: Fase 6 dan "Mesin Payroll Fleksibel" (borongan, uang makan, bonus QC, overhead).*

**Q6.1 — Apa itu "rate card"?** *(P1)*
SRS menyebut "rate card" berisi ongkos potong dan ongkos jahit per artikel/komponen untuk tiap klien. Mohon jelaskan isinya:
- [ ] **A.** Upah yang **dibayar ke penjahit** per pcs
- [ ] **B.** Tarif yang **ditagih ke klien** per pcs
- [ ] **C.** Keduanya (dua kolom berbeda): …

Jika ada ongkos potong, siapa yang menerimanya (tukang potong/cutter)? Apakah dibayar dengan cara yang sama seperti penjahit?

**Jawaban:**

**Q6.2 — Siapa yang berwenang mengubah rate card?** *(P1)*
SRS memberi hak ini kepada **Owner** ("ubah rate card") dan **Kepala Produksi** ("tentukan rate card komponen"). Siapa yang menentukan dan siapa yang hanya mengusulkan? Apakah perubahan perlu persetujuan?

**Jawaban:**

**Q6.3 — Mulai berlakunya perubahan rate card** *(P2)*
Jika rate berubah di tengah minggu, apakah berlaku untuk pekerjaan yang sudah dicatat, atau hanya untuk pekerjaan berikutnya?

**Jawaban:**

**Q6.4 — Siapa penerima bonus QC?** *(P1)*
SRS menyebut bonus dibagi rata kepada **2 Mandor + 1 Kepala Operasional**, tetapi "Kepala Operasional" tidak ada di daftar 8 peran pengguna.
- Siapa (jabatan) yang dimaksud Kepala Operasional? Apakah sama dengan Owner atau Kepala Produksi?
- Jika mandor lebih dari 2 orang, **dua mandor yang mana** yang menerima?
- Apakah penerimanya tetap, atau bergantian?

**Jawaban:**

**Q6.5 — Nilai bonus QC** *(P1)*
- Berapa **rate bonus per pcs** yang lolos QC? Apakah sama untuk semua artikel dan klien?
- Apakah barang yang lolos setelah repair ikut dihitung?
- Kapan bonus dibayarkan (mingguan atau bulanan)?

**Jawaban:**

**Q6.6 — Uang makan (10–15%)** *(P1)*
- Persentase 10–15% dihitung dari total ongkos jahit **siapa** (semua penjahit pada periode itu)?
- Hasilnya **dibagikan ke siapa dan bagaimana**: rata ke semua penjahit, sesuai jumlah hari hadir, atau sesuai output?
- Siapa yang menentukan persentase tiap periode?

**Jawaban:**

**Q6.7 — Overhead per unit** *(P2)*
Biaya bulanan (listrik, air, internet, operasional) dibagi total unit produk. Hasilnya dipakai untuk apa?
- [ ] Hanya untuk laporan biaya
- [ ] Ditambahkan ke harga atau tagihan klien
- [ ] Lainnya: …

Apakah yang dihitung semua unit yang diproduksi, atau hanya yang lolos QC? Siapa yang menginput biaya bulanan?

**Jawaban:**

**Q6.8 — Gaji bulanan** *(P2)*
SRS menyebut Finance menghitung "gaji mingguan **dan bulanan**". Gaji bulanan itu untuk siapa (staf tetap, mandor, Kepala Operasional, lainnya) dan apakah dihitung oleh sistem ini?

**Jawaban:**

**Q6.9 — Penguncian dan koreksi** *(P1)*
SRS menyebut "histori transaksi yang dikunci tidak dapat diubah". Mohon jelaskan:
- Kapan data dianggap terkunci (setelah diverifikasi mandor, setelah gaji Sabtu dihitung, setelah dibayar)?
- Jika ada kesalahan setelah terkunci, bagaimana cara memperbaikinya dan siapa yang berhak?

**Jawaban:**

**Q6.10 — Slip dan pembayaran** *(P2)*
- Apakah sistem perlu menerbitkan **slip upah** untuk penjahit? Contoh bentuk slip saat ini?
- Apakah pembayaran upah (tunai/transfer) dilakukan **di luar sistem**, dan sistem hanya menghasilkan rekap angka?

**Jawaban:**

---

## 7. Finance: Invoice & Pembayaran Klien

*Rujukan SRS: izin peran Finance & Payroll ("terbitkan invoice").*

**Q7.1 — Invoice ke klien** *(P1)*
SRS memberi Finance hak menerbitkan invoice, tetapi tidak ada penjelasan fiturnya. Apakah pembuatan invoice **harus ada di sistem** pada rilis pertama?
- Jika ya: invoice dihitung dari apa (jumlah barang lolos QC × tarif)? Apakah ada format/template yang sudah dipakai?
- Jika tidak: apakah cukup di luar sistem (Excel/aplikasi lain) untuk sementara?

**Jawaban:**

**Q7.2 — Pembayaran dari klien** *(P3)*
Apakah pencatatan pembayaran dari klien (uang muka, pelunasan) perlu masuk sistem?

**Jawaban:**

---

## 8. Barang Jadi, Pengiriman & Penutupan Order

*Rujukan SRS: Fase 5 berhenti di "QC & Barang Jadi", lalu langsung ke Payroll. Tidak ada penjelasan setelah barang lolos QC.*

**Q8.1 — Apa yang terjadi setelah barang lolos QC?** *(P1)*
Mohon jelaskan alurnya sampai order dianggap selesai:
- Bagaimana barang jadi disimpan, dikemas, dan dikirim atau diambil klien?
- Apakah perlu **surat jalan / bukti serah terima** dan siapa yang membuatnya?
- Kapan status order dianggap **"Selesai"**?
- Apakah pengiriman perlu dicatat di sistem pada rilis pertama, atau di luar sistem dahulu?

**Jawaban:**

**Q8.2 — Retur dan komplain** *(P2)*
Jika klien menolak sebagian barang atau mengajukan komplain setelah pengiriman, bagaimana prosesnya? Perlukah dicatat di sistem?

**Jawaban:**

**Q8.3 — Jumlah tidak sesuai target** *(P2)*
Jika hasil lolos QC kurang dari jumlah pesanan klien (karena defect), bagaimana penanganannya? Apakah ada toleransi jumlah (misalnya ±5%)?

**Jawaban:**

---

## 9. Portal Klien (B2B)

*Rujukan SRS: peran "Klien Maklon (B2B Portal)".*

**Q9.1 — Apa yang boleh dilihat klien?** *(P2)*
SRS menyebut klien dapat melihat status pesanan, progres real-time, dan stok bahan titipannya sendiri. Mohon rinci:
- Progres apa yang ditampilkan (persentase, jumlah pcs, tahap)?
- Apakah klien boleh melihat hasil QC, jumlah defect, atau foto produksi?
- Hal apa yang **tidak boleh** dilihat klien (misalnya tarif, data penjahit, klien lain)?

**Jawaban:**

**Q9.2 — Siapa yang mendapat akses dan bagaimana mendapatkannya?** *(P3)*
Berapa kira-kira jumlah klien yang akan memakai portal? Apakah akun dibuatkan oleh Lukla? Apakah satu klien bisa punya lebih dari satu pengguna?

**Jawaban:**

**Q9.3 — Notifikasi** *(P3)*
Apakah klien perlu diberi tahu otomatis (WhatsApp/email) saat ada perubahan status penting?

**Jawaban:**

---

## 10. Owner/Eksekutif: Dashboard & Pengaturan

*Rujukan SRS: peran Owner memiliki "analitik tinggi" dan "konfigurasi sistem".*

**Q10.1 — Laporan yang dibutuhkan Owner** *(P2)*
Istilah "analitik tinggi" belum dijelaskan. Laporan atau angka apa yang ingin Owner lihat (misalnya progres order per klien, produktivitas per line/penjahit, total biaya payroll, efisiensi bahan per klien, jumlah defect)? Seberapa sering dilihat (harian, mingguan)?

**Jawaban:**

**Q10.2 — Pengaturan yang boleh diubah Owner** *(P2)*
Istilah "konfigurasi sistem" belum dijelaskan. Pengaturan apa saja yang Owner perlu ubah sendiri (rate card, persentase uang makan, rate bonus, daftar artikel, kategori cacat, lainnya)?

**Jawaban:**

---

## 11. Pengguna, Perangkat & Lingkungan Kerja

**Q11.1 — Jumlah dan daftar pengguna** *(P2)*
Berapa jumlah pengguna sistem per peran (Owner, Tim Sample, Gudang, Mandor, QC, Finance)? Apakah penjahit juga akan masuk ke sistem (lihat Q4.1)?

**Jawaban:**

**Q11.2 — Perangkat di lantai produksi** *(P1)*
Perangkat apa yang tersedia atau akan disediakan (tablet, HP, komputer) untuk mandor, gudang, QC, dan Finance? Berapa jumlahnya? Apakah dipakai bergantian?

**Jawaban:**

**Q11.3 — Koneksi internet di pabrik** *(P1)*
Apakah internet/Wi-Fi di pabrik Ciwidey stabil di semua area (lantai produksi, gudang, ruang QC)? Apakah pernah putus, dan berapa lama biasanya? Jika terputus, apakah pencatatan tetap harus bisa dilakukan dan dikirim belakangan?

**Jawaban:**

**Q11.4 — Metode login** *(P3)*
Untuk tahap awal kami usulkan login sederhana dengan akun yang disiapkan oleh tim kami (tanpa fitur pendaftaran sendiri). Apakah ini dapat diterima? Adakah kebutuhan khusus (misalnya PIN pendek untuk tablet bersama)?

**Jawaban:**

**Q11.5 — Bahasa** *(P3)*
Apakah antarmuka cukup dalam Bahasa Indonesia?

**Jawaban:**

---

## 12. Data Awal & Contoh Dokumen

Contoh nyata membantu kami memahami cara kerja saat ini dan menyiapkan data awal. **Foto atau pindaian sudah cukup.**

**Q12.1 — Dokumen yang saat ini dipakai** *(P1)*
Mohon kirimkan contoh (boleh disamarkan) dari:
- [ ] Catatan/form output penjahit dan rekap harian
- [ ] Perhitungan gaji borongan mingguan (Excel atau buku)
- [ ] Rate card/daftar tarif yang berlaku
- [ ] Form QC atau catatan cacat
- [ ] Catatan stok gudang dan penerimaan bahan titipan
- [ ] Form sample dan catatan time study
- [ ] Contoh PO bahan, surat jalan, dan invoice
- [ ] Perhitungan bonus QC dan uang makan

**Jawaban / lampiran:**

**Q12.2 — Data awal yang akan dimasukkan** *(P1)*
Daftar berikut perlu disiapkan sebelum sistem dipakai. Mohon konfirmasi apakah sudah ada dan dalam bentuk apa (Excel, buku, belum ada):
- Daftar klien (termasuk Snøhund) dan kontaknya
- Daftar artikel dan komponen
- Rate card per artikel per klien
- Daftar penjahit, mandor, dan staf
- Stok awal bahan (hasil stok opname)
- Order yang sedang berjalan saat sistem mulai dipakai

**Jawaban:**

**Q12.3 — Kondisi saat ini (untuk ukuran keberhasilan)** *(P2)*
Agar kami bisa mengukur apakah sistem benar-benar membantu, mohon perkiraan kondisi **sekarang**:
- Berapa lama menghitung gaji borongan mingguan (dari awal sampai siap dibayar)?
- Seberapa sering bahan klien salah keluar atau tertukar (per bulan/tahun)? Apa dampaknya?
- Berapa lama waktu yang dibutuhkan untuk menelusuri riwayat revisi sample?
- Berapa persen koreksi atau keberatan upah yang biasanya muncul?

**Jawaban:**

---

## 13. Konfirmasi Asumsi dan Batasan Tahap Awal

Berikut asumsi yang kami pakai agar pekerjaan bisa berjalan. Mohon beri tanda **Setuju** atau **Tidak setuju** (dan alasannya).

| No | Asumsi / Batasan | Setuju? | Catatan |
|---|---|---|---|
| A1 | Semua 6 modul masuk rilis awal, dikerjakan bertahap berurutan (gudang dan payroll lebih dulu). | | |
| A2 | Sistem hanya untuk **satu fasilitas (Ciwidey)**. Snøhund diperlakukan sebagai klien internal di sistem yang sama. | | |
| A3 | **Akun pengguna disiapkan tim kami**; belum ada fitur daftar sendiri, undang pengguna, atau reset password di tahap awal. (Data penjahit tetap bisa ditambah lewat aplikasi.) | | |
| A4 | Output produksi **diinput manual** tiap 30 menit; belum ada integrasi mesin/sensor. | | |
| A5 | **Pembayaran upah** dilakukan di luar sistem; sistem hanya menghasilkan rekap dan slip. | | |
| A6 | Belum ada integrasi ke software akuntansi/ERP lain. | | |
| A7 | Belum ada absensi/biometrik. | | |
| A8 | Antarmuka berupa web yang nyaman dipakai di tablet dan komputer; belum ada aplikasi khusus iOS/Android. | | |
| A9 | **Data historis lama tidak dimigrasi.** Hanya data master dan stok awal yang dimasukkan. | | |
| A10 | Jadwal indikatif: kerja 2 mingguan per tahap (±9 tahap, sekitar 4–5 bulan) dengan pilot di 1 line sebelum dipakai penuh. Tahap payroll dijalankan **berdampingan dengan hitungan manual** selama 2–4 minggu. | | |

---

## 14. Hal Lain yang Mungkin Terlewat

**Q14.1** *(P2)* — Adakah proses atau aturan penting di pabrik yang **tidak disebut di SRS**, misalnya pembagian lembur, perencanaan produksi, kasus order dengan beberapa skema sekaligus, pembatalan order, atau pekerjaan oleh pihak ketiga (vendor luar)?

**Jawaban:**

**Q14.2** *(P2)* — Adakah masalah atau kasus kejadian nyata (contoh insiden salah keluar bahan, sengketa upah) yang bisa kami pelajari agar sistem benar-benar menyelesaikannya?

**Jawaban:**

---

*Terima kasih atas waktu dan kerja samanya. Jawaban Anda akan kami pakai untuk menyusun alur proses (diagram BPMN) dan dokumen kebutuhan bisnis (BRD) yang final. Jika ada pertanyaan yang kurang jelas, mohon tanyakan langsung ke kami; lebih baik kami jelaskan daripada Anda menjawab dengan menebak.*
