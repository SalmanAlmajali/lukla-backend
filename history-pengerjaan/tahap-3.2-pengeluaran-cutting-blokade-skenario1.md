# Laporan Pengerjaan Tahap 3.2: Pengeluaran ke Meja Potong, Blokade Kontaminasi Silang (Skenario Uji 1), Log Audit Percobaan Salah-Keluar, & Scrap Cutting (BPMN 02)

**Tanggal Penyelesaian:** 14 Oktober 2026  
**Status:** Selesai (100% Terverifikasi Otomatis)  
**Database Migration:** `000028_create_inventory_outbound_tables.up.sql` (DB Version: 28)  
**Target Modul:** `internal/modules/inventory`, `internal/app`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Tahap 3.2 merupakan implementasi dari alur hilir inventaris bahan baku (*inventory outbound*) menuju meja potong (*cutting table*), pencatatan sisa potongan kain (*scraps*), dan yang paling krusial: **Skenario Uji 1 (Blokade Kontaminasi Silang / Cross-Contamination Blockade)** sesuai ketentuan BPMN 02 dan SRS Dokumen Proyek.

### Aturan Bisnis Utama:
1. **Skenario Uji 1 & FR-6.2 (Cross-Contamination Blockade):**
   - Bahan baku kain yang dibeli untuk klien A **dilarang keras** dipakai memotong pola untuk order klien B.
   - Pada saat pembuatan surat pengeluaran bahan (`POST /api/v1/inventory/issues`), sistem secara otomatis memvalidasi keselarasan kepemilikan:
     $$\text{lot.client\_id} == \text{order.client\_id}$$
   - Apabila terdeteksi ketidakcocokan (*client mismatch*):
     - Transaksi **wajib langsung dibatalkan** (*rollback*) seketika.
     - HTTP response mengembalikan status `400 Bad Request` dengan error code `INVENTORY_CLIENT_ORDER_MISMATCH`.
     - **Saldo stok bahan pada lot sama sekali tidak berkurang** (terjaga utuh).
     - Insiden percobaan salah-ambil bahan dicatat permanen ke dalam tabel audit log `inventory_blocked_attempts` lengkap dengan identitas PIC user gudang, target order, lot kain, dan kronologi alasan.
2. **Pengeluaran Bahan Sah ke Meja Potong:**
   - Menggunakan mekanisme penguncian baris baris (`FOR UPDATE`) untuk memastikan konsistensi saldo di lingkungan concurrent/multi-user.
   - Memotong saldo kuantitas lot secara akurat (`current_quantity = current_quantity - issued_quantity`).
   - Apabila sisa kuantitas lot menjadi 0, status lot otomatis beralih menjadi `DEPLETED`.
3. **Pencatatan Limbah / Sisa Kain Cutting (FR-6.4):**
   - Mendokumentasikan sisa potongan kain pada meja cutting dengan klasifikasi kategori (`PERCA_KECIL`, `CACAT_KAIN_ROLL`, `UJUNG_GULUNGAN`, `KESALAHAN_POTONG`).
   - Menetapkan disposisi limbah (`SIMPAN_GUDANG_SISA`, `KEMBALIKAN_KLIEN`, `JUAL_LIMBAH`, `BUANG`).

---

## 2. Perubahan Skema Database

File migrasi yang diterapkan ke live database PostgreSQL (`lukla_db`):
- `migrations/db/migrations/000028_create_inventory_outbound_tables.up.sql`
- `migrations/db/migrations/000028_create_inventory_outbound_tables.down.sql`

### Rincian Tabel Baru:
1. **`inventory_issues`**:
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `issue_number` (VARCHAR(50), NOT NULL, UNIQUE)
   - `target_order_id` (UUID, NOT NULL, FK `orders(id)`)
   - `cutting_table_number` (VARCHAR(50))
   - `issue_date` (DATE, NOT NULL)
   - `issued_by_user_id` (UUID, NOT NULL, FK `users(id)`)
   - `notes` (TEXT)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
2. **`inventory_issue_items`**:
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `issue_id` (UUID, NOT NULL, FK `inventory_issues(id)` ON DELETE CASCADE)
   - `lot_id` (UUID, NOT NULL, FK `inventory_lots(id)`)
   - `material_id` (UUID, NOT NULL, FK `materials(id)`)
   - `quantity` (NUMERIC(15, 2), CHECK `quantity > 0`)
   - `unit` (VARCHAR(20), NOT NULL)
   - `notes` (TEXT)
   - `created_at` (TIMESTAMPTZ)
3. **`inventory_blocked_attempts` (Audit Trail Skenario Uji 1)**:
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `attempted_at` (TIMESTAMPTZ, NOT NULL, DEFAULT CURRENT_TIMESTAMP)
   - `attempted_by_user_id` (UUID, NOT NULL, FK `users(id)`)
   - `lot_id` (UUID, NOT NULL, FK `inventory_lots(id)`)
   - `lot_number` (VARCHAR(50), NOT NULL)
   - `lot_client_id` (UUID, NOT NULL, FK `clients(id)`)
   - `target_order_id` (UUID, NOT NULL, FK `orders(id)`)
   - `target_order_number` (VARCHAR(50), NOT NULL)
   - `target_client_id` (UUID, NOT NULL, FK `clients(id)`)
   - `requested_quantity` (NUMERIC(15, 2), NOT NULL)
   - `reason` (TEXT, NOT NULL)
4. **`cutting_scraps` (FR-6.4)**:
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `order_id` (UUID, NOT NULL, FK `orders(id)`)
   - `material_id` (UUID, NOT NULL, FK `materials(id)`)
   - `scrap_category` (VARCHAR(50), CHECK IN `PERCA_KECIL`, `CACAT_KAIN_ROLL`, `UJUNG_GULUNGAN`, `KESALAHAN_POTONG`)
   - `quantity` (NUMERIC(15, 2), CHECK `quantity > 0`)
   - `unit` (VARCHAR(20), NOT NULL)
   - `disposition` (VARCHAR(50), CHECK IN `SIMPAN_GUDANG_SISA`, `KEMBALIKAN_KLIEN`, `JUAL_LIMBAH`, `BUANG`)
   - `reported_by_user_id` (UUID, NOT NULL, FK `users(id)`)
   - `notes` (TEXT)
   - `created_at` (TIMESTAMPTZ)

---

## 3. Implementasi Kode & Catatan Rekayasa Kunci

### A. Arsitektur Komponen:
1. **Model (`internal/modules/inventory/model.go`)**:
   - Menambahkan struct request & response: `CreateIssueRequest`, `CreateIssueItem`, `IssueSummary`, `IssueDetail`, `IssueItemPayload`, `BlockedAttemptRecord`, `ListIssuesFilter`, `ListBlockedAttemptsFilter`, `CreateScrapRequest`, `CuttingScrapRecord`, `ListScrapsFilter`.
   - Menambahkan konstanta error: `ErrCrossContaminationBlocked`, `ErrInsufficientStock`, `ErrLotNotAvailable`, `ErrIssueNotFound`, `ErrIssueItemsEmpty`, `ErrScrapCategoryInvalid`, `ErrScrapDispositionInvalid`.
2. **Repository (`internal/modules/inventory/repository.go` & `postgres_repository.go`)**:
   - `CreateIssue`: Menjalankan validasi kepemilikan client antara lot dan target order. Menggunakan `tx.Rollback()` seketika saat mismatch terdeteksi sebelum mencatat audit attempt agar tidak terjadi *foreign-key lock wait* di PostgreSQL, lalu mengembalikan `ErrCrossContaminationBlocked`.
   - `FindIssueByID`, `ListIssues`: Query detail dan daftar surat pengeluaran bahan.
   - `RecordBlockedAttempt`, `ListBlockedAttempts`: Penyimpanan dan pembacaan log audit upaya salah-keluar yang diblokir.
   - `CreateScrap`, `ListScraps`: Pencatatan dan pembacaan limbah cutting kain.
3. **Service Layer (`internal/modules/inventory/service.go`)**:
   - Validasi RBAC ketat:
     - `CreateIssue`, `GetIssue`, `ListIssues`, `ListBlockedAttempts`: Dapat diakses oleh `WAREHOUSE` dan `OWNER`.
     - `CreateScrap`, `ListScraps`: Dapat diakses oleh `PRODUCTION`, `WAREHOUSE`, dan `OWNER`.
4. **Handler Layer (`internal/modules/inventory/handler.go`)**:
   - Registrasi endpoint:
     - `POST /api/v1/inventory/issues`
     - `GET /api/v1/inventory/issues`
     - `GET /api/v1/inventory/issues/:id`
     - `GET /api/v1/inventory/blocked-attempts`
     - `POST /api/v1/inventory/scraps`
     - `GET /api/v1/inventory/scraps`
   - Mapping error `ErrCrossContaminationBlocked` ke HTTP status `400 Bad Request` dengan error code `INVENTORY_CLIENT_ORDER_MISMATCH`.

---

## 4. Hasil Pengujian Otomatis

### A. Acceptance Test Live Database (`internal/app/app_test.go`)
Fungsi uji `TestApp_InventoryOutboundAndBlockadeLiveDB` memverifikasi 5 titik kritis operasional pabrik:
1. **Skenario Uji 1 (Cross-Contamination Blockade)**:
   - Mencoba mengeluarkan kain milik klien Aventura Outdoor Gear (`LOT-AVE-20261014-01`) untuk SPK Snøhund Brand (`PO-2026-10-001`).
   - **Hasil**: Sistem seketika menolak dengan HTTP `400 Bad Request` dan kode error `INVENTORY_CLIENT_ORDER_MISMATCH`.
   - **Verifikasi Integritas Stok**: Saldo stok lot Aventura dipastikan tetap utuh 100.00 KG (sama sekali tidak terpotong).
   - **Verifikasi Audit Trail**: Log audit `inventory_blocked_attempts` terisi otomatis mencatat insiden pemblokiran.
2. **Skenario Pengeluaran Sah (Valid Issue)**:
   - Mengeluarkan 50.0 KG kain Snøhund (`LOT-SNO-20261014-01`) untuk SPK Snøhund (`PO-2026-10-001`).
   - **Hasil**: HTTP `201 Created`, nomor surat pengeluaran terbit, saldo sisa lot terpotong menjadi 100.5 KG.
3. **Verifikasi Detail & Daftar Pengeluaran**:
   - Memanggil `GET /api/v1/inventory/issues/:id` dan `GET /api/v1/inventory/issues`.
   - **Hasil**: HTTP `200 OK` dengan relasi item dan remaining quantity yang akurat.
4. **Pencatatan Limbah Cutting (Scrap)**:
   - Memanggil `POST /api/v1/inventory/scraps` untuk 4.25 KG `PERCA_KECIL` dengan disposisi `SIMPAN_GUDANG_SISA`.
   - **Hasil**: HTTP `201 Created` dan verifikasi `GET /api/v1/inventory/scraps` menghasilkan HTTP `200 OK`.

### B. Hasil Eksekusi Unit Test Seluruh Backend
```bash
go test ./...
```
Output:
```
ok      github.com/lukla/operations-engine/internal/app         17.328s
ok      github.com/lukla/operations-engine/internal/modules/identity    (cached)
ok      github.com/lukla/operations-engine/internal/modules/inventory   (cached)
ok      github.com/lukla/operations-engine/internal/modules/masterdata  (cached)
ok      github.com/lukla/operations-engine/internal/modules/payroll     (cached)
ok      github.com/lukla/operations-engine/internal/modules/production  (cached)
ok      github.com/lukla/operations-engine/internal/modules/ratecard    (cached)
PASS (All test suites passed 100%)
```

---

## 5. Ringkasan Endpoint & Contoh Response Riil

### 1. `POST /api/v1/inventory/issues` - Skenario Uji 1: Blokade Kontaminasi Silang
- **HTTP Status:** `400 Bad Request`
- **Contoh Request:**
  ```json
  {
      "targetOrderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
      "cuttingTableNumber": "MEJA-CUTTING-01",
      "issuedDate": "2026-10-15",
      "notes": "Percobaan salah ambil kain milik Aventura untuk SPK Snøhund",
      "items": [
          {
              "lotId": "f075a84d-ac2a-4b7d-a688-7328c4c03927",
              "quantity": 25.0,
              "notes": "Salah ambil roll kain milik brand klien lain"
          }
      ]
  }
  ```
- **Contoh Response Riil:**
  ```json
  {
      "error": {
          "code": "INVENTORY_CLIENT_ORDER_MISMATCH",
          "message": "BLOKIR KEAMANAN: Bahan tidak dapat dikeluarkan karena pemilik lot bahan berbeda dengan klien order tujuan: BLOKIR KEAMANAN: Bahan pada lot 'LOT-AVE-20261014-01' milik klien 'Aventura Outdoor Gear' tidak dapat dikeluarkan untuk order 'PO-2026-10-001' milik klien 'Snøhund Brand'."
      },
      "success": false
  }
  ```

### 2. `GET /api/v1/inventory/blocked-attempts` - Audit Trail Percobaan Salah-Keluar
- **HTTP Status:** `200 OK`
- **Contoh Response Riil:**
  ```json
  {
      "data": [
          {
              "attemptedAt": "2026-10-14T23:09:18.050193+07:00",
              "attemptedByName": "Agus Pratama",
              "attemptedByUserId": "bfe9c52d-769d-4645-b621-06ad7aa0c716",
              "id": "86c76d10-ae5e-40b6-a21e-fd3f2737b908",
              "lotClientId": "483b04f2-d5b3-4581-97e9-27ccbfb38637",
              "lotClientName": "Aventura Outdoor Gear",
              "lotId": "f075a84d-ac2a-4b7d-a688-7328c4c03927",
              "lotNumber": "LOT-AVE-20261014-01",
              "reason": "BLOKIR KEAMANAN: Bahan pada lot 'LOT-AVE-20261014-01' milik klien 'Aventura Outdoor Gear' tidak dapat dikeluarkan untuk order 'PO-2026-10-001' milik klien 'Snøhund Brand'.",
              "requestedQuantity": 25,
              "targetClientId": "f310f933-4b79-40c4-849d-f0334716d43c",
              "targetClientName": "Snøhund Brand",
              "targetOrderCode": "PO-2026-10-001",
              "targetOrderId": "fad2530f-90d9-4a76-af96-d9f7db451a98"
          }
      ],
      "success": true,
      "total": 1
  }
  ```

### 3. `POST /api/v1/inventory/issues` - Pengeluaran Bahan Valid ke Cutting
- **HTTP Status:** `201 Created`
- **Contoh Response Riil:**
  ```json
  {
      "data": {
          "createdAt": "2026-10-14T23:09:18.084282+07:00",
          "cuttingTableNumber": "MEJA-CUTTING-01",
          "id": "7a347ac6-b1ba-4084-8240-e0500c0abb3f",
          "issueNumber": "ISS-20261014-0003",
          "issuedByName": "Agus Pratama",
          "issuedDate": "2026-10-15",
          "items": [
              {
                  "clientName": "Snøhund Brand",
                  "id": "2310bb2a-1fba-4bf3-a74c-a018ab53e8b0",
                  "lotId": "4c2591b6-a60b-4f97-b50e-f8c936e1fd57",
                  "lotNumber": "LOT-SNO-20261014-01",
                  "materialCode": "MAT-CTN-30S-BLK",
                  "materialName": "Cotton Combed 30s Reaktif - Hitam Jet Black",
                  "notes": "Ambil 50kg",
                  "quantity": 50,
                  "remainingLotQuantity": 100.5,
                  "unit": "KG"
              }
          ],
          "notes": "Pengeluaran bahan kain badan Snøhund sesuai SPK",
          "targetClientName": "Snøhund Brand",
          "targetOrderCode": "PO-2026-10-001",
          "targetOrderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
          "totalItemsCount": 1
      },
      "message": "Bahan berhasil dikeluarkan ke bagian cutting untuk order PO-2026-10-001",
      "success": true
  }
  ```

### 4. `GET /api/v1/inventory/issues` - Daftar Pengeluaran Bahan
- **HTTP Status:** `200 OK`
- **Contoh Response Riil:**
  ```json
  {
      "data": [
          {
              "createdAt": "2026-10-14T23:09:18.084282+07:00",
              "cuttingTableNumber": "MEJA-CUTTING-01",
              "id": "7a347ac6-b1ba-4084-8240-e0500c0abb3f",
              "issueNumber": "ISS-20261014-0003",
              "issuedDate": "2026-10-15T00:00:00Z",
              "targetClientName": "Snøhund Brand",
              "targetOrderCode": "PO-2026-10-001",
              "targetOrderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
              "totalItemsCount": 1
          }
      ],
      "success": true,
      "total": 1
  }
  ```

### 5. `GET /api/v1/inventory/issues/:id` - Detail Pengeluaran Bahan
- **HTTP Status:** `200 OK`
- **Contoh Response Riil:**
  ```json
  {
      "data": {
          "createdAt": "2026-10-14T23:09:18.084282+07:00",
          "cuttingTableNumber": "MEJA-CUTTING-01",
          "id": "7a347ac6-b1ba-4084-8240-e0500c0abb3f",
          "issueNumber": "ISS-20261014-0003",
          "issuedByName": "Agus Pratama",
          "issuedDate": "2026-10-15T00:00:00Z",
          "items": [
              {
                  "clientName": "Snøhund Brand",
                  "id": "2310bb2a-1fba-4bf3-a74c-a018ab53e8b0",
                  "lotId": "4c2591b6-a60b-4f97-b50e-f8c936e1fd57",
                  "lotNumber": "LOT-SNO-20261014-01",
                  "materialCode": "MAT-CTN-30S-BLK",
                  "materialName": "Cotton Combed 30s Reaktif - Hitam Jet Black",
                  "notes": "Ambil 50kg",
                  "quantity": 50,
                  "remainingLotQuantity": 100.5,
                  "unit": "KG"
              }
          ],
          "notes": "Pengeluaran bahan kain badan Snøhund sesuai SPK",
          "targetClientName": "Snøhund Brand",
          "targetOrderCode": "PO-2026-10-001",
          "targetOrderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
          "totalItemsCount": 1
      },
      "success": true
  }
  ```

### 6. `POST /api/v1/inventory/scraps` - Pencatatan Limbah Cutting
- **HTTP Status:** `201 Created`
- **Contoh Response Riil:**
  ```json
  {
      "data": {
          "createdAt": "2026-10-14T23:09:18.139211+07:00",
          "disposition": "SIMPAN_GUDANG_SISA",
          "id": "bc4eba1e-b2a3-41a9-8f39-4ea026932f66",
          "materialId": "a3150c7e-fd5e-4ca1-bf6f-b4a148c0186d",
          "materialName": "Cotton Combed 30s Reaktif - Hitam Jet Black",
          "notes": "Sisa pemotongan pola badan dan kerah batch 1",
          "orderCode": "PO-2026-10-001",
          "orderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
          "quantity": 4.25,
          "recordedByName": "Bambang Supriyadi",
          "scrapCategory": "PERCA_KECIL",
          "unit": "KG"
      },
      "message": "Data sisa potongan cutting berhasil dicatat",
      "success": true
  }
  ```

### 7. `GET /api/v1/inventory/scraps` - Daftar Limbah Cutting
- **HTTP Status:** `200 OK`
- **Contoh Response Riil:**
  ```json
  {
      "data": [
          {
              "createdAt": "2026-10-14T23:09:18.139211+07:00",
              "disposition": "SIMPAN_GUDANG_SISA",
              "id": "bc4eba1e-b2a3-41a9-8f39-4ea026932f66",
              "materialId": "a3150c7e-fd5e-4ca1-bf6f-b4a148c0186d",
              "materialName": "Cotton Combed 30s Reaktif - Hitam Jet Black",
              "notes": "Sisa pemotongan pola badan dan kerah batch 1",
              "orderCode": "PO-2026-10-001",
              "orderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
              "quantity": 4.25,
              "recordedByName": "Bambang Supriyadi",
              "scrapCategory": "PERCA_KECIL",
              "unit": "KG"
          }
      ],
      "success": true,
      "total": 1
  }
  ```

---

## 6. Sinkronisasi Postman Collection

File `Lukla_Articles_and_Components.postman_collection.json` telah diperbarui dengan folder:
**`09. Tahap 3: Gudang & Inventaris - Penerimaan, Stok Lot, Pengeluaran Cutting & Blokade Silang (BPMN 02)`**

Item baru yang ditambahkan:
- **Item 10**: `POST /inventory/issues - Blokade Kontaminasi Silang (Skenario Uji 1 / 400 Bad Request)`
- **Item 11**: `GET /inventory/blocked-attempts - Audit Trail Percobaan Salah-Keluar (200 OK)`
- **Item 12**: `POST /inventory/issues - Pengeluaran Bahan Valid ke Cutting (201 Created)`
- **Item 13**: `GET /inventory/issues - Daftar Surat Pengeluaran Bahan (200 OK)`
- **Item 14**: `GET /inventory/issues/:id - Detail Surat Pengeluaran Bahan (200 OK)`
- **Item 15**: `POST /inventory/scraps - Pencatatan Limbah Cutting / Scrap (201 Created)`
- **Item 16**: `GET /inventory/scraps - Daftar Scrap Cutting (200 OK)`

Variabel koleksi baru:
- `lot_aventura_id`: `f075a84d-ac2a-4b7d-a688-7328c4c03927`
- `issue_id`: `7a347ac6-b1ba-4084-8240-e0500c0abb3f`
- `scrap_id`: `bc4eba1e-b2a3-41a9-8f39-4ea026932f66`

