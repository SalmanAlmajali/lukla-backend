INSERT INTO roles (code, name, description)
VALUES
    (
        'owner_executive',
        'Owner / Eksekutif',
        'Cakupan: Internal. Izin: Analitik tinggi & konfigurasi sistem, Ubah rate card. Pembatasan: Tidak dapat mengubah histori transaksi yang dikunci.'
    ),
    (
        'sample_adis',
        'Tim Sample / Adis',
        'Cakupan: Sample. Izin: Catat pemakaian bahan sample, Input time study per komponen, Setujui versi sample. Pembatasan: Tidak dapat menyetujui pesanan massal tanpa konfirmasi.'
    ),
    (
        'warehouse',
        'Gudang / Warehouse',
        'Cakupan: Stok. Izin: Terima bahan + titipan klien, Keluarkan bahan ke cutting, Stok opname. Pembatasan: Dilarang mengeluarkan bahan jika ID Klien/Pesanan tidak cocok.'
    ),
    (
        'production_head',
        'Kepala Produksi / Mandor',
        'Cakupan: Line. Izin: Atur line + alokasi penjahit, Tentukan rate card komponen, Catat output 30 menit. Pembatasan: Tidak dapat mengubah formula penggajian atau hasil QC final.'
    ),
    (
        'tailor',
        'Penjahit / Tailor',
        'Cakupan: Operator. Izin: Lihat tugas komponen, Laporkan hasil kerja (diverifikasi mandor). Pembatasan: Tidak dapat mengedit catatan output yang sudah diverifikasi.'
    ),
    (
        'qc_inspector',
        'QC Inspector',
        'Cakupan: Quality. Izin: Periksa barang jadi, Set status lolos/reject/repair, Catat jenis cacat. Pembatasan: Tidak dapat mengubah harga borongan atau data order klien.'
    ),
    (
        'finance_payroll',
        'Finance & Payroll',
        'Cakupan: Finance. Izin: Buat/setujui PO bahan, Terbitkan invoice, Hitung gaji mingguan & bulanan. Pembatasan: Tidak dapat mengubah fisik stok di gudang.'
    ),
    (
        'client',
        'Klien Maklon / External Client',
        'Cakupan: B2B Portal. Izin: Pantau status pesanan, Lihat progres real-time, Lihat stok bahan titipan sendiri. Pembatasan: Terbatas hanya pada data client_id milik sendiri.'
    );

