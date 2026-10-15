# Laporan Pengerjaan Tahap 5.2: Order Completion Lifecycle & Notification Engine (BPMN 11 & 01)

**Tanggal Penyelesaian:** 15 Oktober 2026  
**Status:** Selesai (100% PASS - Unit Tests, Acceptance Tests Live DB, & Full Test Suite)  
**Dokumen Referensi Perencanaan:** [analisis-dan-rencana-tahap-5.md](./analisis-dan-rencana-tahap-5.md)  
**Spesifikasi OpenAPI:** [openapi-phase5-portal-klien.yaml](../openapi-phase5-portal-klien.yaml)  
**Database Migration:** Versi 31 (`000031_create_notifications_and_order_completion.up.sql`)  
**Modul Baru Engine:** `internal/modules/notification`, `internal/modules/order`  
**Modul yang Disempurnakan:** `internal/modules/quality`, `internal/modules/inventory`, `internal/modules/portal`, `internal/app`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Berdasarkan diagram alur bisnis **BPMN 11 (Client Tracking & Notification)** dan bagian penutup **BPMN 01 (End-to-End Maklon Garment Flow)**, terdapat kebutuhan mendasar untuk:
1. **Otomasi Siklus Hidup Pesanan Selesai (Order Completion):**
   - Ketika kuantitas barang yang lolos inspeksi QC mencapai atau melampaui target pesanan (`passed_quantity >= target_quantity`), status order pabrik harus otomatis bertransisi dari `IN_PRODUCTION` menjadi `COMPLETED`.
   - Sistem mencatat waktu resmi penyelesaian pesanan pada kolom `completed_at` (audit timestamp).
2. **Penyelesaian Manual & Verifikasi Akhir (Manual Completion):**
   - Pemilik pabrik (*Owner / Executive*) dan Kepala Produksi (*Production Head*) memiliki wewenang untuk menutup dan memverifikasi penyelesaian pesanan secara manual melalui API `POST /orders/:id/complete`.
3. **In-App Notification Engine & Event Dispatcher:**
   - Klien maklon memerlukan sistem notifikasi in-app otomatis untuk mengabarkan kejadian penting:
     - `ORDER_COMPLETED`: Pesanan telah selesai diproduksi dan seluruh garmen siap dikirim / diserahterimakan.
     - `MATERIAL_RECEIVED`: Bahan baku kain/aksesoris titipan klien telah sampai dan berhasil dicatat di gudang pabrik.
     - `ANNOUNCEMENT`: Pengumuman resmi dari staf pabrik (misal jadwal ekspedisi atau penyerahan barang).
4. **Isolasi Multi-Tenant pada Notifikasi Klien:**
   - Klien A hanya dapat membaca dan menandai notifikasi yang terikat pada `client_id` miliknya. Upaya akses silang antar-klien otomatis ditolak dengan status `404 Not Found` (*anti-IDOR & anti-data leakage*).
   - Pengguna internal pabrik non-client ditolak dengan `403 Forbidden` saat mencoba mengakses rute portal notifikasi.

---

## 2. Perubahan Skema Database (Migration 31)

Diterapkan file migrasi PostgreSQL live:
- `migrations/db/migrations/000031_create_notifications_and_order_completion.up.sql`
- `migrations/db/migrations/000031_create_notifications_and_order_completion.down.sql`

### Rincian Perubahan Skema:
1. **Kolom Baru pada Tabel `orders`:**
   ```sql
   ALTER TABLE orders ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
   ```
2. **Tabel Baru `notifications`:**
   ```sql
   CREATE TABLE IF NOT EXISTS notifications (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
       title VARCHAR(255) NOT NULL,
       message TEXT NOT NULL,
       event_type VARCHAR(50) NOT NULL, -- 'ORDER_COMPLETED', 'MATERIAL_RECEIVED', 'PRODUCTION_MILESTONE', 'ANNOUNCEMENT'
       reference_id UUID,
       is_read BOOLEAN NOT NULL DEFAULT FALSE,
       read_at TIMESTAMPTZ,
       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
   );

   CREATE INDEX IF NOT EXISTS idx_notifications_client_id ON notifications(client_id);
   CREATE INDEX IF NOT EXISTS idx_notifications_client_unread ON notifications(client_id, is_read);
   CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);
   ```

---

## 3. Arsitektur & Komponen Kode

### A. Modul Baru `internal/modules/notification`
- `model.go`: Tipe event (`ORDER_COMPLETED`, `MATERIAL_RECEIVED`, dll), entity `Notification`, DTO request/response.
- `repository.go` & `postgres_repository.go`: Penyimpanan notifikasi, filter tenant, query unread count, mark-read dengan validasi ownership.
- `service.go`: Business logic pembuatan dan penandaan baca notifikasi.
- `handler.go`: Rute portal (`/portal/notifications`, `/portal/notifications/:id/read`, `/portal/notifications/read-all`) dan rute internal (`/notifications/send`).
- `notification_test.go`: Unit test suite komprehensif mock repository (`PASS`).

### B. Modul Baru `internal/modules/order`
- `model.go`: DTO pesanan internal pabrik (`InternalOrderSummary`, `InternalOrderDetail`, `CompleteOrderRequest`).
- `repository.go` & `postgres_repository.go`: Query SQL daftar order internal pabrik, detail dengan breakdown cutting, sewing, QC, dan update `COMPLETED` dengan `completed_at`.
- `service.go`: Logika penutupan pesanan dengan pemicu otomatis pengiriman notifikasi `ORDER_COMPLETED` ke klien pemilik order.
- `handler.go`: Rute internal pabrik (`GET /orders`, `GET /orders/:id`, `POST /orders/:id/complete`).
- `order_test.go`: Unit test suite alur penutupan order dan verifikasi pemicu notifikasi (`PASS`).

### C. Modul yang Disempurnakan
- `internal/modules/quality`: Pada kueri `CreateInspection`, ketika akumulasi garmen lolos QC mencapai target pesanan (`cumPassed >= targetQty`), status order otomatis diubah menjadi `COMPLETED`, `completed_at = NOW()`, dan menerbitkan notifikasi `ORDER_COMPLETED` ke klien pemilik pesanan.
- `internal/modules/inventory`: Pada transaksi penerimaan bahan (`CreateReceipt`), secara otomatis menerbitkan notifikasi `MATERIAL_RECEIVED` ke klien pemilik lot bahan baku.
- `internal/modules/portal`: Pada `GetDashboard`, metrik `unreadNotificationsCount` kini menghitung jumlah riil notifikasi belum dibaca dari tabel `notifications`.
- `internal/app`: Pemasangan dependensi dan registrasi rute `notification` serta `order` pada router Gin engine.

---

## 4. Spesifikasi & Daftar Endpoint

| No | Method | Path | Deskripsi | Akses Role | Status Code |
|---|---|---|---|---|---|
| 1 | `POST` | `/notifications/send` | Kirim notifikasi/pengumuman manual ke akun klien | Internal Pabrik (`owner_executive`, `production_head`) | `201 Created`, `403 Forbidden` |
| 2 | `GET` | `/portal/notifications` | Mengambil riwayat notifikasi in-app dan total belum dibaca (`unreadCount`) | Klien Portal (`client`) | `200 OK` |
| 3 | `PUT` | `/portal/notifications/:id/read` | Menandai notifikasi spesifik sebagai sudah dibaca (`is_read = true`) | Klien Portal (`client`) | `200 OK`, `404 Not Found` |
| 4 | `PUT` | `/portal/notifications/read-all` | Menandai seluruh notifikasi belum dibaca menjadi sudah dibaca | Klien Portal (`client`) | `200 OK` |
| 5 | `GET` | `/orders` | Mengambil seluruh daftar pesanan pabrik (filter `status`, `clientId`) | Internal Pabrik | `200 OK` |
| 6 | `GET` | `/orders/:id` | Rincian teknis lengkap pesanan internal pabrik | Internal Pabrik | `200 OK`, `404 Not Found` |
| 7 | `POST` | `/orders/:id/complete` | Menutup status pesanan menjadi `COMPLETED` dan memicu notifikasi ke klien | Internal Pabrik (`owner_executive`, `production_head`) | `200 OK`, `400 Bad Request` |

---

## 5. Bukti Pengujian Otomatis

### A. Unit Tests Modul
```
=== RUN   TestNotificationService_Flow
--- PASS: TestNotificationService_Flow (0.00s)
PASS
ok  	github.com/lukla/operations-engine/internal/modules/notification	3.525s

=== RUN   TestOrderService_CompleteOrderFlow
--- PASS: TestOrderService_CompleteOrderFlow (0.00s)
PASS
ok  	github.com/lukla/operations-engine/internal/modules/order	2.231s
```

### B. Live Database Acceptance Test (`TestApp_OrderCompletionAndNotificationsLiveDB`)
Diuji langsung pada database live PostgreSQL (`localhost:5433/lukla_db`) mencakup 10 skenario pengujian:
```
=== RUN   TestApp_OrderCompletionAndNotificationsLiveDB
    app_test.go:4360: 
        === ACTUAL POST /api/v1/notifications/send ===
        {
            "data": {
                "id": "b13378a4-9953-4fc6-985a-3ac1fb61ddc0",
                "clientId": "f310f933-4b79-40c4-849d-f0334716d43c",
                "title": "Pemberitahuan Jadwal Pengiriman Batch 1",
                "message": "Pengiriman 3.336 pcs Parka Arctic Ekspor dijadwalkan pada hari Jumat pukul 09:00 WIB.",
                "eventType": "ANNOUNCEMENT",
                "isRead": false,
                "createdAt": "2026-10-15T08:44:56.934832+07:00"
            },
            "message": "Notifikasi berhasil dikirim",
            "success": true
        }
    app_test.go:4383: 
        === ACTUAL GET /api/v1/portal/notifications ===
        {
            "data": [
                {
                    "id": "b13378a4-9953-4fc6-985a-3ac1fb61ddc0",
                    "clientId": "f310f933-4b79-40c4-849d-f0334716d43c",
                    "title": "Pemberitahuan Jadwal Pengiriman Batch 1",
                    "message": "Pengiriman 3.336 pcs Parka Arctic Ekspor dijadwalkan pada hari Jumat pukul 09:00 WIB.",
                    "eventType": "ANNOUNCEMENT",
                    "isRead": false,
                    "createdAt": "2026-10-15T08:44:56.934832+07:00"
                }
            ],
            "success": true,
            "unreadCount": 1
        }
    app_test.go:4405: Dashboard Unread Notifications Count: 1
    app_test.go:4419: Cross-tenant notification read isolation protected: 404 Not Found
    app_test.go:4458: 
        === ACTUAL GET /api/v1/orders ===
        {
            "data": [
                {
                    "id": "15eb181f-4ed8-4176-9f16-814f59bbb735",
                    "orderNumber": "PO-URBAN-999",
                    "clientId": "b303bd49-7608-44a2-af22-3d57d88b487b",
                    "clientName": "Urban Culture Wear",
                    "articleId": "e084fffd-2b54-4027-85b5-7bfc1a4070d1",
                    "articleCode": "ART-SNO-PK01",
                    "articleTitle": "Snøhund Arctic Expedition Parka 2026",
                    "targetQuantity": 500,
                    "passedQuantity": 0,
                    "status": "IN_PRODUCTION",
                    "createdAt": "2026-10-15T08:10:31.311778+07:00"
                },
                {
                    "id": "a9142b01-3c14-4d2d-bb1f-7c9b1ac64fa5",
                    "orderNumber": "PO-2026-10-002",
                    "clientId": "483b04f2-d5b3-4581-97e9-27ccbfb38637",
                    "clientName": "Aventura Outdoor Gear",
                    "articleId": "7f418075-d444-4059-9f73-7b28d2c85b46",
                    "articleCode": "ART-AVE-TR01",
                    "articleTitle": "Aventura All-Terrain Cargo Pants",
                    "targetQuantity": 500,
                    "passedQuantity": 0,
                    "status": "IN_PRODUCTION",
                    "startDate": "2026-10-14",
                    "targetDate": "2026-11-04",
                    "createdAt": "2026-10-14T17:30:25.310537+07:00"
                },
                {
                    "id": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                    "orderNumber": "PO-2026-10-001",
                    "clientId": "f310f933-4b79-40c4-849d-f0334716d43c",
                    "clientName": "Snøhund Brand",
                    "articleId": "e084fffd-2b54-4027-85b5-7bfc1a4070d1",
                    "articleCode": "ART-SNO-PK01",
                    "articleTitle": "Snøhund Arctic Expedition Parka 2026",
                    "targetQuantity": 1000,
                    "passedQuantity": 3392,
                    "status": "COMPLETED",
                    "startDate": "2026-10-14",
                    "targetDate": "2026-11-13",
                    "createdAt": "2026-10-14T17:30:25.310537+07:00"
                }
            ],
            "success": true
        }
    app_test.go:4479: 
        === ACTUAL GET /api/v1/orders/:id ===
        {
            "data": {
                "id": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                "orderNumber": "PO-2026-10-001",
                "clientId": "f310f933-4b79-40c4-849d-f0334716d43c",
                "clientName": "Snøhund Brand",
                "articleId": "e084fffd-2b54-4027-85b5-7bfc1a4070d1",
                "articleCode": "ART-SNO-PK01",
                "articleTitle": "Snøhund Arctic Expedition Parka 2026",
                "targetQuantity": 1000,
                "passedQuantity": 3392,
                "status": "COMPLETED",
                "startDate": "2026-10-14",
                "targetDate": "2026-11-13",
                "createdAt": "2026-10-14T17:30:25.310537+07:00",
                "notes": "Pesanan Ekspor Snohund Arctic Parka Batch 1",
                "cuttingQuantity": 1000,
                "sewingQuantity": 3725,
                "repairQuantity": 14,
                "rejectQuantity": 14,
                "passRate": 99.2
            },
            "success": true
        }
    app_test.go:4508: 
        === ACTUAL POST /api/v1/orders/:id/complete ===
        {
            "data": {
                "id": "a9142b01-3c14-4d2d-bb1f-7c9b1ac64fa5",
                "orderNumber": "PO-2026-10-002",
                "clientId": "483b04f2-d5b3-4581-97e9-27ccbfb38637",
                "clientName": "Aventura Outdoor Gear",
                "articleId": "7f418075-d444-4059-9f73-7b28d2c85b46",
                "articleCode": "ART-AVE-TR01",
                "articleTitle": "Aventura All-Terrain Cargo Pants",
                "targetQuantity": 500,
                "passedQuantity": 0,
                "status": "COMPLETED",
                "startDate": "2026-10-14",
                "targetDate": "2026-11-04",
                "completedAt": "2026-10-15T08:44:57.042748+07:00",
                "createdAt": "2026-10-14T17:30:25.310537+07:00",
                "notes": "Pesanan Aventura telah diverifikasi QC lolos 100% dan siap diambil klien.",
                "cuttingQuantity": 500,
                "sewingQuantity": 0,
                "repairQuantity": 0,
                "rejectQuantity": 0,
                "passRate": 0
            },
            "message": "Pesanan berhasil ditandai selesai dan notifikasi telah dikirim ke klien",
            "success": true
        }
    app_test.go:4519: Verified: Automatic ORDER_COMPLETED notification created for client order (count: 1)
--- PASS: TestApp_OrderCompletionAndNotificationsLiveDB (0.70s)
PASS
ok  	github.com/lukla/operations-engine/internal/app	4.073s
```

### C. Full Test Suite Engine (`go test ./...`)
Seluruh modul engine lulus uji 100% tanpa regresi:
```
ok  	github.com/lukla/operations-engine/internal/app	21.972s
ok  	github.com/lukla/operations-engine/internal/modules/identity	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/inventory	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/masterdata	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/notification	4.849s
ok  	github.com/lukla/operations-engine/internal/modules/order	3.849s
ok  	github.com/lukla/operations-engine/internal/modules/payroll	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/portal	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/production	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/qcbonus	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/quality	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/ratecard	(cached)
```

---

## 6. Sinkronisasi Postman Collection

Berkas [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) telah diperbarui dengan folder baru:
`13. Tahap 5.2: Order Completion & Notification Engine (BPMN 11 & 01)`

Daftar request pengujian otomatis yang disertakan:
1. `01. POST /notifications/send - Kirim Notifikasi Manual ke Klien (201 Created)`
2. `02. GET /portal/notifications - Daftar Notifikasi Milik Klien (200 OK)`
3. `03. PUT /portal/notifications/:id/read - Tandai Notifikasi Sudah Dibaca (200 OK)`
4. `04. PUT /portal/notifications/:id/read - Uji Keamanan Multi-Tenant Akses Silang (404 Not Found)`
5. `05. PUT /portal/notifications/read-all - Tandai Seluruh Notifikasi Dibaca (200 OK)`
6. `06. GET /orders - Daftar Seluruh Pesanan Pabrik Internal (200 OK)`
7. `07. GET /orders/:id - Detail Pesanan Internal Pabrik (200 OK)`
8. `08. POST /orders/:id/complete - Selesaikan Pesanan & Trigger Notif Klien (200 OK)`
9. `09. POST /orders/:id/complete - Uji RBAC Klien Dilarang Selesaikan Order (403 Forbidden)`
10. `10. POST /notifications/send - Uji RBAC Klien Dilarang Kirim Notifikasi (403 Forbidden)`

