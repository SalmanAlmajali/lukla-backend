# Laporan Pengerjaan Tahap 3.1: Katalog Bahan, Penerimaan Bertagging (Inbound Receipts), & Saldo Stok Lot (BPMN 02)

**Tanggal Penyelesaian:** 14 Oktober 2026  
**Status:** Selesai (100% Verified)  
**Versi Migrasi Database:** 000026 & 000027  
**OpenAPI Specification:** [`openapi-phase3-gudang.yaml`](../openapi-phase3-gudang.yaml)  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Mengacu pada diagram alur operasional `Lukla_BPMN_v2.drawio` (**Diagram 2: Penerimaan Bahan & Blokade Kontaminasi Silang**) dan Software Requirements Specification (SRS):
- **FR-6.1 (Mandatory Tagging Inbound):** Bahan baku yang masuk ke gudang (kain, rib, benang, zipper) **wajib langsung ditandai (*tagged*)** ke Klien Pemilik (`client_id`) dan Surat Perintah Kerja / Order (`order_id`). Sistem tidak boleh menerima bahan tak bertuan (*unassigned/orphan stock*).
- **Inbound Cross-Check Validation:** Jika pengguna memasukkan `order_id` yang terdaftar atas nama klien lain, sistem harus seketika menolak transaksi penerimaan (*mismatch rejection*) dengan status `400 Bad Request` dan kode error `INVENTORY_CLIENT_ORDER_MISMATCH`.
- **Granularitas Stok Berbasis Lot (Roll / Batch):** Setiap roll kain yang diterima dari supplier dicatat ke entitas `inventory_lots` mandiri dengan nomor lot (`lot_number`), kuantitas awal, kuantitas tersedia (`available_quantity`), lokasi simpan (`location`), dan status (`AVAILABLE`).
- **Isolasi Multi-Tenant Klien:** Klien maklon (role `client`) yang mengakses endpoint inventaris hanya diizinkan melihat tanda terima dan lot bahan yang dialokasikan khusus untuk brand mereka.

---

## 2. Perubahan Skema & Seeding Database (Migration 000026 & 000027)

Dibuat berkas migrasi database PostgreSQL:
1. **Migration 000026 (DDL Schema):**
   - **Up Migration:** `migrations/db/migrations/000026_create_inventory_inbound_tables.up.sql`
   - **Down Migration:** `migrations/db/migrations/000026_create_inventory_inbound_tables.down.sql`
2. **Migration 000027 (Data Seeding Inbound & Lots):**
   - **Up Migration:** `migrations/db/migrations/000027_seed_inventory_lots.up.sql`
   - **Down Migration:** `migrations/db/migrations/000027_seed_inventory_lots.down.sql`
   - **Isi Seeding:** 
     - Surat Jalan `RCV-20261014-0001` untuk Snøhund (`PO-2026-10-001`) dengan 2 roll kain Cotton Combed 30s Hitam (Total 300 KG di `GUDANG-A-01` & `GUDANG-A-02`).
     - Surat Jalan `RCV-20261014-0002` untuk Aventura (`PO-2026-10-002`) dengan 1 roll kain Cotton Combed 30s Putih (Total 100 KG di `GUDANG-B-01`).

### Tabel Baru yang Dibuat:
1. **`materials`**: Master katalog bahan baku pabrik (kategori `FABRIC`, `ACCESSORY`, `THREAD`, dsb., unit bawaan `METER`, `KG`, `PCS`).
2. **`inventory_receipts`**: Header surat jalan & tanda terima gudang bertagging `client_id` dan `order_id`, dicatat oleh petugas penerima (`received_by_user_id`).
3. **`inventory_lots`**: Detail fisik roll/batch bahan dengan pelacakan saldo sisa (`available_quantity`), lokasi rak penyimpanan, dan status lot (`AVAILABLE`, `DEPLETED`, `RESERVED`).
4. **`inventory_receipt_items`**: Catatan per item pada surat penerimaan yang langsung ditautkan dengan lot yang tercipta.

### Index & Integrity Constraints:
- Foreign key cascading/restricting pada `client_id`, `order_id`, `material_id`, dan `received_by_user_id`.
- Indeks performa pada `materials(code)`, `inventory_receipts(client_id, order_id, receipt_number)`, dan `inventory_lots(material_id, client_id, order_id, lot_number, status)`.
- Seeding 3 bahan awal: Kain Taslan Milky Waterproof (`MAT-FAB-TASLAN-01`), Polar Fleece 280 GSM (`MAT-FAB-FLEECE-01`), dan Zipper YKK Vislon #5 (`MAT-ACC-ZIP-YKK5`).

---

## 3. Komponen Kode & Arsitektur

Modul baru dibangun dengan Clean Architecture di `internal/modules/inventory/`:
- **`model.go`**: Definisi domain structs (`Material`, `Receipt`, `ReceiptItem`, `Lot`, `MaterialStockSummary`), DTO request/response, dan konstanta error (`ErrClientOrderMismatch`, `ErrMaterialNotFound`, `ErrReceiptNotFound`, `ErrLotNotFound`).
- **`repository.go`**: Interface repository abstraksi untuk database access.
- **`postgres_repository.go`**: Implementasi PostgreSQL menggunakan transaksi atomik (`BeginTx`). Menggabungkan validasi kepemilikan order-client, insert header receipt, generasi nomor penerimaan otomatis (`RCV-YYYYMMDD-XXXX`), insert lots, dan item receipt. Juga mengimplementasikan query agregasi saldo stok (`ListStockBalances`) dan filter multi-tenant.
- **`service.go`**: Business layer yang mengurus otorisasi RBAC (role `warehouse`, `owner_executive`, `production_head`, dan `client`), penegakan aturan mandatory tagging, dan proteksi multi-tenant context.
- **`handler.go`**: HTTP controller Gin untuk handling serialization, query parameters parsing, dan response formatting standar.
- **`internal/app/app.go`**: Registrasi route grup `/api/v1/inventory/*` dengan middleware autentikasi JWT dan otorisasi role.

---

## 4. Endpoints yang Diimplementasikan

| Method | Endpoint | Role Akses | Deskripsi & Validasi |
|---|---|---|---|
| `GET` | `/api/v1/inventory/materials` | Warehouse, Owner, Prod Head, Client | Katalog bahan baku aktif |
| `POST` | `/api/v1/inventory/materials` | Warehouse, Owner | Tambah master bahan baru |
| `GET` | `/api/v1/inventory/materials/:id` | Warehouse, Owner, Prod Head, Client | Rincian spesifikasi 1 bahan |
| `POST` | `/api/v1/inventory/receipts` | Warehouse, Owner | **Penerimaan bahan bertagging**. Validasi ketat `order.client_id == receipt.client_id`. Otomatis membuat lot dan item penerimaan |
| `GET` | `/api/v1/inventory/receipts` | Warehouse, Owner, Prod Head, Client | Riwayat surat penerimaan. Multi-tenant terfilter jika diakses role `client` |
| `GET` | `/api/v1/inventory/receipts/:id` | Warehouse, Owner, Prod Head, Client | Detail surat penerimaan dan lot yang dihasilkan |
| `GET` | `/api/v1/inventory/stocks` | Warehouse, Owner, Prod Head, Client | Rekapitulasi saldo stok per bahan dan rincian lot yang tersedia |
| `GET` | `/api/v1/inventory/lots/:id` | Warehouse, Owner, Prod Head, Client | Inspeksi 1 barcode/lot untuk traceability dan tagging audit |

---

## 5. Bukti Pengujian Otomatis

### A. Unit Tests (`internal/modules/inventory`)
18 unit test pada `internal/modules/inventory/handler_test.go` mencakup pengujian sukses, kegagalan validasi, dan isolasi role:
```
=== RUN   TestHandler_CreateMaterial
--- PASS: TestHandler_CreateMaterial (0.00s)
=== RUN   TestHandler_CreateReceipt_Success
--- PASS: TestHandler_CreateReceipt_Success (0.00s)
=== RUN   TestHandler_CreateReceipt_MismatchClientOrder
--- PASS: TestHandler_CreateReceipt_MismatchClientOrder (0.00s)
=== RUN   TestHandler_ListStocks_Warehouse
--- PASS: TestHandler_ListStocks_Warehouse (0.00s)
=== RUN   TestHandler_ListStocks_ClientIsolation
--- PASS: TestHandler_ListStocks_ClientIsolation (0.00s)
...
PASS
ok      github.com/lukla/operations-engine/internal/modules/inventory   6.465s
```

### B. Live DB Acceptance Test (`internal/app`)
Pengujian end-to-end terintegrasi dengan database PostgreSQL riil pada `TestApp_InventoryInboundLiveDB` (`internal/app/app_test.go`):
1. **Otentikasi:** Petugas Gudang (`warehouse@lukla.local`), Eksekutif (`executive@lukla.local`), dan Klien (`client@lukla.local`).
2. **Katalog Bahan:** Petugas gudang berhasil membuat material baru via `POST /api/v1/inventory/materials`.
3. **Uji Penolakan Mismatch (FR-6.1):** Mengirim `POST /api/v1/inventory/receipts` dengan kombinasi `clientId` Klien A dan `orderId` Klien B. Sistem secara tepat menolak dengan status `400 Bad Request` dan pesan `INVENTORY_CLIENT_ORDER_MISMATCH`.
4. **Penerimaan Valid & Multi-Lot:** Mengirim surat jalan supplier dengan 2 roll bahan bertagging valid (`order_id` & `client_id`). Response `201 Created` mengembalikan receipt number dan ID lot masing-masing roll.
5. **Agregasi Saldo Stok:** `GET /api/v1/inventory/stocks` memverifikasi total saldo material bertambah secara matematis sesuai penjumlahan kuantitas kedua lot.
6. **Verifikasi Multi-Tenant:** Klien mengakses inventaris; hanya data yang dialokasikan ke klien bersangkutan yang tampil, mengamankan privasi inventaris antar-brand.
```
=== RUN   TestApp_InventoryInboundLiveDB
--- PASS: TestApp_InventoryInboundLiveDB (0.57s)
PASS
ok      github.com/lukla/operations-engine/internal/app 5.526s
```

### C. Full Project Regression Suite
Semua unit dan integration test di seluruh modul backend lulus 100%:
```
ok      github.com/lukla/operations-engine/internal/app         16.590s
ok      github.com/lukla/operations-engine/internal/modules/identity    (cached)
ok      github.com/lukla/operations-engine/internal/modules/inventory   6.465s
ok      github.com/lukla/operations-engine/internal/modules/masterdata  (cached)
ok      github.com/lukla/operations-engine/internal/modules/payroll     (cached)
ok      github.com/lukla/operations-engine/internal/modules/production  (cached)
ok      github.com/lukla/operations-engine/internal/modules/ratecard    (cached)
```

---

## 6. Pembaruan Postman Collection

File `Lukla_Articles_and_Components.postman_collection.json` telah diperbarui dengan penambahan folder baru:
- **Nama Folder:** `09. Tahap 3: Gudang & Inventaris - Penerimaan & Stok Lot (BPMN 02)`
- **Jumlah Endpoint:** 9 request interaktif lengkap
- **Otomasi Pengujian (`pm.test`):** Assertion status HTTP, validasi format data, dan penyimpanan otomatis token gudang, `material_id`, `receipt_id`, dan `lot_id` ke variabel koleksi.
- **Contoh Respon Lengkap:** Setiap request dilengkapi contoh respon riil status `200 OK` / `201 Created` beserta payload body JSON lengkap.

---

## 7. Rencana Tindak Lanjut (Menuju Tahap 3.2)

Dengan rampungnya Tahap 3.1, fondasi inventaris bahan baku dan stok lot bertagging telah siap. Tahap selanjutnya adalah **Tahap 3.2: Pengeluaran ke Meja Potong, Blokade Kontaminasi Silang (Skenario Uji 1), Audit Log, & Scrap Cutting**:
1. Pembuatan tabel `inventory_issues` (pengeluaran bahan) dan `inventory_issue_items`.
2. Pembuatan tabel `security_audit_logs` / `blocked_contamination_attempts` untuk mencatat upaya pengeluaran bahan salah lot (Skenario Uji 1 / FR-6.3).
3. Pembuatan tabel `cutting_scraps` untuk pelacakan perca dan sisa kain potong (FR-6.4).
4. Implementasi logika blokade: validasi ketat apakah lot yang hendak dipotong (`lot.client_id` dan `lot.order_id`) sesuai dengan SPK meja potong tujuan.

