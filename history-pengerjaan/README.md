# History Pengerjaan Backend Lukla Operations Engine

Folder ini berisi dokumentasi dan catatan riwayat pengerjaan setiap tahapan pengembangan backend sesuai dengan roadmap pada `urutan pengerjaan.txt` dan diagram proses bisnis `Lukla_BPMN_v2.drawio`.

---

## Tabel Riwayat & Status Pengerjaan

| Tahap | Modul & BPMN | Tanggal Selesai | DB Migration | Status | Dokumen Laporan |
|---|---|---|---|---|---|
| **Tahap 0** | Fondasi & Master Data (BPMN 10) | Sebelumnya | 000001 - 000018 | Selesai | [Lihat OpenApi Phase 0](../openapi-phase0-fondasi.yaml) |
| **Tahap 1** | Produksi & Terminal Mandor (BPMN 04, 05) | Sebelumnya | 000019 - 000023 | Selesai | [Lihat OpenApi Phase 1](../openapi-phase1-produksi.yaml) |
| **Tahap 2.1** | Payroll Core Engine & Uji Skenario 2 (BPMN 07) | 14 Okt 2026 | 000024 | Selesai | [Laporan Tahap 2.1](./tahap-2.1-payroll-periode-kalkulasi-skenario2.md) |
| **Tahap 2.2** | Locking, Audit Adjustment, & Summary Report (BPMN 07) | 14 Okt 2026 | 000025 | Selesai | [Laporan Tahap 2.2](./tahap-2.2-payroll-locking-adjustment-summary.md) |
| **Tahap 3.1** | Katalog Bahan, Penerimaan Bertagging, & Stok Lot (BPMN 02) | 14 Okt 2026 | 000026 - 000027 | Selesai | [Laporan Tahap 3.1](./tahap-3.1-katalog-bahan-penerimaan-stok-lot.md) |
| **Tahap 3.2** | Pengeluaran ke Cutting & Blokade Kontaminasi Silang (BPMN 02) | 14 Okt 2026 | 000028 | Selesai | [Laporan Tahap 3.2](./tahap-3.2-pengeluaran-cutting-blokade-skenario1.md) |
| **Tahap 4.1** | QC Engine, Defect Tracking, & Verifikasi Upah Penjahit (BPMN 06) | 15 Okt 2026 | 000029 | Selesai | [Laporan Tahap 4.1](./tahap-4.1-qc-inspeksi.md) |
| **Tahap 4.2** | Pool Bonus QC 3-Arah & Skenario Uji 3 (BPMN 08) | 15 Okt 2026 | 000030 | Selesai | [Laporan Tahap 4.2](./tahap-4.2-pool-bonus-skenario3.md) |
| **Tahap 5.1** | Portal Klien & Isolasi Multi-Tenant (BPMN 11) | 15 Okt 2026 | - | Selesai | [Laporan Tahap 5.1](./tahap-5.1-portal-klien.md) |
| **Tahap 5.2** | Order Completion Lifecycle & Notification Engine (BPMN 11, 01) | 15 Okt 2026 | 000031 | Selesai | [Laporan Tahap 5.2](./tahap-5.2-order-completion-notifikasi.md) |
| **Tahap 6.1** | Sample Lifecycle, Versioning Engine, & Time Study Tracker (BPMN 03) | 15 Okt 2026 | 000032 | Selesai | [Laporan Tahap 6.1](./tahap-6.1-sample-time-study.md) |
| **Tahap 6.2** | Commercial Rate Card, Client Approval, Order Conversion & Skenario Uji 4 (BPMN 03, 11) | *Berikutnya* | 000032 | Direncanakan | [Rencana Tahap 6](./analisis-dan-rencana-tahap-6.md#b-tahap-62-commercial-rate-card-client-approval-portal-konversi-ke-order-massal--skenario-uji-4-bpmn-03--11) |
| **Tahap 7** | Overhead & Uji End-to-End (BPMN 09, 01) | *Akan Datang* | - | Direncanakan | - |

---

## Standar Kelengkapan Setiap Laporan Tahap

Setiap laporan tahap yang ditambahkan ke folder ini wajib mencakup:
1. **Latar Belakang & Kebutuhan Bisnis**: Masalah operasional pabrik yang diselesaikan dan aturan bisnis (*business rules*) yang diakomodasi.
2. **Perubahan Skema Database**: Detail migrasi PostgreSQL (DDL up/down, foreign key, index, dan constraint checks).
3. **Arsitektur & Komponen Kode**: Rincian berkas yang dibuat/diubah pada layer model, repository, service, dan HTTP handler.
4. **Spesifikasi & Contoh Request/Response Endpoint**: Payload JSON, HTTP status code, dan skenario error.
5. **Bukti Pengujian Otomatis**:
   - Hasil Unit Test modul.
   - Hasil Acceptance Test Live Database (termasuk verifikasi skenario uji SRS).
6. **Sinkronisasi Postman Collection**: Konfirmasi bahwa kumpulan endpoint pada file `Lukla_Articles_and_Components.postman_collection.json` telah diperbarui dengan test scripts otomatis (`pm.test`) dan contoh response riil.

