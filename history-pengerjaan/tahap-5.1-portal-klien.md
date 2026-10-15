# Laporan Pengerjaan Tahap 5.1: Portal Klien & Isolasi Multi-Tenant (BPMN 11)

**Tanggal Penyelesaian:** 15 Oktober 2026  
**Status:** Selesai (100% PASS - Unit Tests & Live Database Acceptance Tests)  
**Dokumen Referensi Perencanaan:** [analisis-dan-rencana-tahap-5.md](./analisis-dan-rencana-tahap-5.md)  
**Spesifikasi OpenAPI:** [openapi-phase5-portal-klien.yaml](../openapi-phase5-portal-klien.yaml)  
**Modul Engine:** `internal/modules/portal`  
**Router API:** `/api/v1/portal/*`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Berdasarkan diagram proses bisnis **BPMN 11 (Client Tracking & Notification)** dan dokumen SRS maklon garmen Lukla, pemilik brand/klien maklon memerlukan transparansi penuh terhadap:
1. **Status Progres Pesanan (Order Tracking):** Klien ingin mengetahui sejauh mana pesanan mereka diproses di pabrik tanpa harus menghubungi pihak admin melalui WhatsApp secara manual.
2. **Breakdown 3-Tahap Produksi:** Klien dapat melihat progres pesanan terbagi dalam 3 tahapan kunci:
   - **Tahap 1 - Pemotongan (Cutting):** Estimasi panel pola kain yang telah dipotong dari gelondongan bahan.
   - **Tahap 2 - Penjahitan & Perakitan (Sewing / Assembly):** Jumlah akumulasi output rakitan penjahit line produksi.
   - **Tahap 3 - Quality Control (QC Lolos):** Jumlah garmen jadi yang telah diperiksa oleh tim QC dan dinyatakan lolos inspeksi (`PASSED`).
3. **Transparansi Mutu & Cacat (QC Transparency):** Klien dapat memantau tingkat kelolosan (*pass rate*), jumlah barang yang perlu diperbaiki (*repair*), dan cacat permanen (*reject*).
4. **Saldo Bahan Baku Titipan Klien (Consigned Materials):** Klien dapat memantau sisa kuantitas kain dan aksesoris milik mereka yang masih tersimpan di gudang pabrik per lot penerimaan dan lokasi rak/gudang.
5. **Saldo Barang Jadi Siap Kirim (Finished Goods):** Klien dapat memantau barang jadi yang sudah lolos inspeksi akhir dan siap untuk diambil/dikirim (*Ready for Dispatch*).
6. **Multi-Tenant Data Isolation & Anti-IDOR (Strict Security):** Pabrik menangani banyak brand klien kompetitor (misalnya Snøhund Brand dan Urban Culture). Sistem wajib menjamin tidak ada kebocoran data (*zero cross-tenant data leakage*). Klien A tidak boleh mengakses detail order, stok bahan, maupun laporan mutu milik Klien B.

---

## 2. Perubahan Skema Database

Tahap 5.1 mengonsolidasikan data dari tabel-tabel operasional yang sudah ada (`clients`, `orders`, `articles`, `assemblies`, `qc_inspections`, `materials`, `inventory_lots`) dengan agregasi performan tinggi. Tidak ada perubahan DDL baru yang merusak skema yang sudah ada (*zero destructive migration*). Seluruh relasi memanfaatkan foreign keys yang telah dibuat pada Tahap 0 hingga Tahap 4.

Key query indexing yang dioptimalkan:
- `orders(client_id, status)`
- `assemblies(order_id)`
- `qc_inspections(order_id)`
- `inventory_lots(client_id, current_quantity, status)`

---

## 3. Arsitektur & Komponen Kode

Modul baru domain Go diimplementasikan di `internal/modules/portal`:

| Berkas | Fungsi Utama |
|---|---|
| `model.go` | Mendefinisikan DTO respon portal: `ClientPortalDashboard`, `ClientPortalOrderSummary`, `ClientPortalOrderDetail`, `ClientPortalOrderQuality`, `ClientPortalMaterialStock`, `ClientPortalLotStock`, dan `ClientPortalFinishedGoodsStock`. |
| `repository.go` | Interface kontrak abstraksi data storage: `GetDashboard`, `ListOrders`, `GetOrderDetail`, `GetOrderQuality`, `ListMaterialStocks`, `ListFinishedGoodsStocks`. |
| `postgres_repository.go` | Implementasi PostgreSQL repository dengan kueri agregasi SQL, filtering tenant ID aman, dan perlindungan kepemilikan. |
| `service.go` | Layer business logic untuk validasi keberadaan `clientID` pada konteks pengguna, validasi isolasi tenant, dan kalkulasi persentase penyelesaian. |
| `handler.go` | HTTP controller Gin untuk registrasi rute, parsing query filter/pagination, dan penanganan respon JSON standar Lukla (`success: true/false`). |
| `portal_test.go` | Unit test suite dengan mock repository untuk menguji semua logika service dan aturan penolakan hak akses. |

### Wiring Router (`internal/app/app.go`)
```go
// Wire Portal Klien (Tahap 5.1 - BPMN 11)
portalRepo := portal.NewPostgresRepository(db)
portalService := portal.NewService(portalRepo)
portalHandler := portal.NewHandler(portalService)
portal.RegisterRoutes(v1, portalHandler, authnMiddleware)
```

---

## 4. Mekanisme Keamanan Multi-Tenant & Anti-IDOR

1. **Dekripsi Identitas dari JWT:**
   - Parameter `clientID` **tidak pernah** diambil dari URL path ataupun query string klien.
   - Handler mengekstrak `actor.ClientID` yang didekripsi langsung dari tanda tangan JWT token oleh middleware autentikasi.
2. **Penolakan Non-Client Role (RBAC):**
   - Jika pengguna internal pabrik (Operator Jahit, Supervisor Line, Mandor) mencoba memanggil endpoint portal, sistem langsung menolak dengan `403 Forbidden` (`FORBIDDEN_CLIENT_PORTAL`).
3. **Pencegahan Enumerasi ID (Anti-IDOR):**
   - Jika Klien A mencoba mengakses order ID milik Klien B melalui `GET /api/v1/portal/orders/:id`, kueri database memfilter `WHERE o.id = $1 AND o.client_id = $2`.
   - Hasil kueri mengembalikan 0 baris sehingga handler mengembalikan `404 Not Found` (`ORDER_NOT_FOUND`). Hal ini mencegah pihak luar menebak keberadaan ID order di database (*anti-enumeration protection*).

---

## 5. Spesifikasi & Daftar Endpoint

Semua endpoint berada di bawah namespace `/api/v1/portal/*` dan memerlukan header `Authorization: Bearer <client_token>`:

| No | Method | Path | Deskripsi | Status Code |
|---|---|---|---|---|
| 1 | `GET` | `/portal/dashboard` | Ringkasan profil klien, jumlah order aktif/selesai, sisa stok bahan, dan barang jadi siap kirim | `200 OK`, `403 Forbidden` |
| 2 | `GET` | `/portal/orders` | Daftar pesanan milik klien dengan filter `status` (`ALL`, `DRAFT`, `IN_PRODUCTION`, `COMPLETED`), search, dan pagination | `200 OK` |
| 3 | `GET` | `/portal/orders/:id` | Detail pesanan dengan progres 3-tahap (Cutting, Sewing, QC Lolos) | `200 OK`, `404 Not Found` |
| 4 | `GET` | `/portal/orders/:id/quality` | Laporan transparansi mutu QC: total diperiksa, lolos, repair, reject, dan pass rate (%) | `200 OK`, `404 Not Found` |
| 5 | `GET` | `/portal/stocks/materials` | Saldo stok lot bahan baku titipan klien di gudang pabrik | `200 OK` |
| 6 | `GET` | `/portal/stocks/finished-goods` | Saldo barang jadi yang sudah lolos QC dan berstatus `READY_FOR_DISPATCH` | `200 OK` |

---

## 6. Bukti Pengujian Otomatis

### A. Unit Test Suite (`portal_test.go`)
Semua skenario business logic service diuji dengan mock repository:
```
=== RUN   TestPortalService_GetDashboard
--- PASS: TestPortalService_GetDashboard (0.00s)
=== RUN   TestPortalService_ListOrders
--- PASS: TestPortalService_ListOrders (0.00s)
=== RUN   TestPortalService_GetOrderDetail
--- PASS: TestPortalService_GetOrderDetail (0.00s)
=== RUN   TestPortalService_GetOrderQuality
--- PASS: TestPortalService_GetOrderQuality (0.00s)
=== RUN   TestPortalService_ListMaterialStocks
--- PASS: TestPortalService_ListMaterialStocks (0.00s)
=== RUN   TestPortalService_ListFinishedGoodsStocks
--- PASS: TestPortalService_ListFinishedGoodsStocks (0.00s)
PASS
ok  	github.com/lukla/operations-engine/internal/modules/portal	3.385s
```

### B. Live Database Acceptance Test (`TestApp_ClientPortalLiveDB`)
Diuji langsung pada database live PostgreSQL (`localhost:5433/lukla_db`) dengan 8 skenario integrasi menyeluruh:

```
=== RUN   TestApp_ClientPortalLiveDB
    app_test.go:4145: 
        === ACTUAL GET /api/v1/portal/dashboard ===
        {
            "data": {
                "client": {
                    "id": "f310f933-4b79-40c4-849d-f0334716d43c",
                    "name": "Snøhund Brand",
                    "code": "SNOHUND"
                },
                "activeOrdersCount": 0,
                "completedOrdersCount": 1,
                "readyFinishedGoodsCount": 3336,
                "activeMaterialsCount": 1,
                "unreadNotificationsCount": 0
            },
            "success": true
        }
    app_test.go:4175: 
        === ACTUAL GET /api/v1/portal/orders ===
        {
            "data": [
                {
                    "id": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                    "orderNumber": "PO-2026-10-001",
                    "articleName": "Snøhund Arctic Expedition Parka 2026",
                    "articleCode": "ART-SNO-PK01",
                    "category": "GARMENT",
                    "targetQuantity": 1000,
                    "passedQuantity": 3336,
                    "progressPercentage": 100,
                    "status": "COMPLETED",
                    "targetDate": "2026-11-13",
                    "createdAt": "2026-10-14T17:30:25.310537+07:00"
                }
            ],
            "pagination": {
                "limit": 10,
                "page": 1,
                "total": 1,
                "totalPages": 1
            },
            "success": true
        }
    app_test.go:4209: 
        === ACTUAL GET /api/v1/portal/orders/:id ===
        {
            "data": {
                "id": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                "orderNumber": "PO-2026-10-001",
                "article": {
                    "id": "e084fffd-2b54-4027-85b5-7bfc1a4070d1",
                    "code": "ART-SNO-PK01",
                    "name": "Snøhund Arctic Expedition Parka 2026",
                    "category": "GARMENT"
                },
                "targetQuantity": 1000,
                "status": "COMPLETED",
                "startDate": "2026-10-14",
                "targetDate": "2026-11-13",
                "notes": "Pesanan Ekspor Snohund Arctic Parka Batch 1",
                "progress": {
                    "cuttingQuantity": 1000,
                    "sewingQuantity": 3650,
                    "passedQCQuantity": 3336,
                    "repairQCQuantity": 12,
                    "rejectQCQuantity": 12,
                    "overallPercentage": 100
                }
            },
            "success": true
        }
    app_test.go:4255: 
        === ACTUAL GET /api/v1/portal/orders/:id/quality ===
        {
            "data": {
                "orderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                "orderNumber": "PO-2026-10-001",
                "targetQuantity": 1000,
                "totalChecked": 3360,
                "passedQuantity": 3336,
                "repairQuantity": 12,
                "rejectQuantity": 12,
                "passRate": 99.3,
                "inspectionsCount": 13
            },
            "success": true
        }
    app_test.go:4279: 
        === ACTUAL GET /api/v1/portal/stocks/materials ===
        {
            "data": [
                {
                    "materialId": "a3150c7e-fd5e-4ca1-bf6f-b4a148c0186d",
                    "materialCode": "MAT-CTN-30S-BLK",
                    "materialName": "Cotton Combed 30s Reaktif - Hitam Jet Black",
                    "category": "FABRIC",
                    "defaultUnit": "KG",
                    "totalAvailableQuantity": 300,
                    "lots": [
                        {
                            "lotId": "4c2591b6-a60b-4f97-b50e-f8c936e1fd57",
                            "lotNumber": "LOT-SNO-20261014-01",
                            "availableQuantity": 150.5,
                            "unit": "KG",
                            "location": "GUDANG-A-01",
                            "receivedDate": "2026-10-14"
                        },
                        {
                            "lotId": "1378a140-97b3-4dd6-a128-868fd7dced3a",
                            "lotNumber": "LOT-SNO-20261014-02",
                            "availableQuantity": 149.5,
                            "unit": "KG",
                            "location": "GUDANG-A-02",
                            "receivedDate": "2026-10-14"
                        }
                    ]
                }
            ],
            "success": true
        }
    app_test.go:4294: 
        === ACTUAL GET /api/v1/portal/stocks/finished-goods ===
        {
            "data": [
                {
                    "orderId": "fad2530f-90d9-4a76-af96-d9f7db451a98",
                    "orderNumber": "PO-2026-10-001",
                    "articleCode": "ART-SNO-PK01",
                    "articleName": "Snøhund Arctic Expedition Parka 2026",
                    "readyQuantity": 3336,
                    "status": "READY_FOR_DISPATCH"
                }
            ],
            "success": true
        }
--- PASS: TestApp_ClientPortalLiveDB (0.88s)
PASS
ok  	github.com/lukla/operations-engine/internal/app	5.350s
```

### C. Full Test Suite (`go test ./...`)
Semua 18 modul dan package pada engine backend berhasil lulus uji 100% tanpa regresi:
```
ok  	github.com/lukla/operations-engine/internal/app	22.845s
ok  	github.com/lukla/operations-engine/internal/modules/identity	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/inventory	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/masterdata	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/payroll	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/portal	3.385s
ok  	github.com/lukla/operations-engine/internal/modules/production	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/qcbonus	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/quality	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/ratecard	(cached)
```

---

## 7. Sinkronisasi Postman Collection

Berkas [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) telah diperbarui dengan folder baru:
`12. Tahap 5.1: Portal Klien & Isolasi Multi-Tenant (BPMN 11)`

Rincian request yang disertakan:
1. `01. POST /auth/login - Login sebagai Akun Klien Snøhund (Simpan client_token)`
2. `02. GET /portal/dashboard - Ringkasan Portal Klien (200 OK)`
3. `03. GET /portal/orders - Daftar Pesanan Klien (200 OK)`
4. `04. GET /portal/orders/:id - Detail Pesanan & Progres 3-Tahap (200 OK)`
5. `05. GET /portal/orders/:id - Uji Keamanan Multi-Tenant Akses Silang Klien Lain (404 Not Found)`
6. `06. GET /portal/orders/:id/quality - Laporan Mutu QC Pesanan Klien (200 OK)`
7. `07. GET /portal/stocks/materials - Saldo Stok Bahan Baku Titipan Klien (200 OK)`
8. `08. GET /portal/stocks/finished-goods - Saldo Barang Jadi Siap Kirim (200 OK)`
9. `09. GET /portal/dashboard - Uji RBAC Staf Non-Client Akses Portal (403 Forbidden)`

Seluruh URL terkonfigurasi dengan tepat menggunakan `{{base_url}}/portal/...` tanpa duplikasi `api/v1`.

