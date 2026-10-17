# Laporan Pengerjaan Tahap 6.2: Commercial Rate Card, Client Approval Portal, Konversi Order Massal & Pembuktian Skenario Uji 4 (BPMN 03 & 11)

**Tanggal:** 15 Oktober 2026  
**Status:** SELESAI 100% (Terverifikasi di Live Database)  
**Referensi:**
- Roadmap: `urutan pengerjaan.txt` (Tahap 6: BPMN 03, Skenario Uji 4)
- Diagram Proses Bisnis: `Lukla_BPMN_v2.drawio` (Diagram 3: Sample ke Order Massal & Diagram 11: Portal Klien)
- Rencana Kerja: [analisis-dan-rencana-tahap-6.md](./analisis-dan-rencana-tahap-6.md)
- Spesifikasi Kontrak API: [openapi-phase6-sample-time-study.yaml](../openapi-phase6-sample-time-study.yaml)
- Koleksi Postman: [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) (Folder 15)

---

## 1. Latar Belakang & Kebutuhan Bisnis

Tahap 6.2 merupakan mata rantai krusial yang menghubungkan proses eksperimen prototipe fisik di pabrik dengan persetujuan komersial brand klien dan penerbitan Purchase Order (PO) produksi massal ribuan potong pakaian:

1. **Commercial Client Rate Card (BPMN 03 & 11):**
   - Berbeda dengan *Internal Labor Rate Card* (upah borongan operator jahit per komponen), Commercial Rate Card adalah **tarif jasa maklon CMT (*Cut, Make, Trim*) per potong baju** yang ditagihkan pabrik kepada brand klien (contoh: Rp 45.000 / pcs untuk Jaket Parka Snøhund).
   - Tarif ini ditentukan secara rasional oleh manajemen pabrik berdasarkan total Standard Allowed Minute (SAM) hasil observasi time study stopwatch versi sample terkait dan target margin laba kotor.
2. **Client Approval Portal (BPMN 11 & BPMN 03):**
   - Brand klien eksternal dapat masuk ke portal klien secara mandiri untuk melihat daftar pengajuan sample, meninjau foto/spesifikasi fitting, melihat riwayat revisi, dan memeriksa penawaran harga CMT.
   - **Request Revision:** Klien dapat meminta revisi teknis pola/aksesoris jika sample fisik belum memuaskan (memicu status `REVISED` dan notifikasi in-app otomatis `SAMPLE_REVISION_REQUESTED` ke tim pabrik).
   - **Digital Approval:** Klien memberikan persetujuan resmi digital (`APPROVED`) pada versi sample final yang disepakati (memicu notifikasi `SAMPLE_APPROVED` dan mengunci acuan spesifikasi teknis).
3. **Pembuktian Mutlak Skenario Uji 4 (SRS Skenario 4):**
   - Skenario Uji 4 pada SRS menetapkan aturan absolut:
     > *"Sample v1 direvisi ke v2, v2 disetujui, dan order massal merujuk spesifikasi yang benar."*
   - Saat sample yang berstatus `APPROVED` dikonversi menjadi order produksi massal:
     - Sistem menjamin pesanan massal tersebut **secara definitif mengikat `sample_version_id` versi v2 yang disetujui**, bukan versi v1 yang sudah usang/cacat desain.
     - Sistem secara otomatis mengadopsi `proposed_unit_price` dari versi v2 (misal Rp 45.000) dan menghitung nilai kontrak:
       $$\text{total\_amount} = \text{target\_quantity} \times \text{unit\_price} = 2.000 \times \text{Rp } 45.000 = \text{Rp } 90.000.000$$
     - Kolom `converted_order_id` pada `sample_requests` terisi ID order massal, dan status sample terkunci menjadi `CONVERTED_TO_ORDER`.
     - Standard SAM pada masterdata `components` otomatis disinkronkan dari hasil observasi time study versi v2 final.
     - Upaya re-konversi ulang (*duplicate conversion*) diblokir dengan aman (*idempotency protection*).
     - Isolasi multi-tenant anti-IDOR mencegah klien brand lain mengintip atau menyetujui sample milik klien yang bukan haknya.

---

## 2. Perubahan Skema Database

Skema database PostgreSQL telah dipersiapkan sejak Migrasi Versi 32 (`000032_create_samples_and_time_study_tables.up.sql`) sehingga tidak memerlukan migrasi DDL baru:
- `orders.sample_version_id UUID REFERENCES sample_versions(id) ON DELETE SET NULL`
- `orders.unit_price NUMERIC(15,2)`
- `orders.total_amount NUMERIC(15,2)`
- `sample_requests.converted_order_id UUID REFERENCES orders(id) ON DELETE SET NULL`
- `sample_versions.proposed_unit_price NUMERIC(15,2)`, `currency`, `price_notes`, `approved_at`, `client_feedback`

---

## 3. Arsitektur & Komponen Kode yang Diperbarui

### A. Modul Sample (`internal/modules/sample`)
- **[`model.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/sample/model.go):**
  - Menambahkan struct `ConvertSampleToOrderPayload` dan `ConvertedMassOrderResponse`.
- **[`repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/sample/repository.go) & [`postgres_repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/sample/postgres_repository.go):**
  - Mengimplementasikan `ConvertToOrder(ctx, sampleID, payload)`: Memvalidasi sample `APPROVED`, memastikan belum pernah dikonversi, mengunci versi yang disetujui, menyalin harga satuan CMT, menghitung `total_amount`, menerbitkan baris baru di tabel `orders`, menyinkronkan Standard SAM ke masterdata `components`, dan memperbarui `sample_requests.converted_order_id`.
- **[`service.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/sample/service.go) & [`handler.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/sample/handler.go):**
  - Mendaftarkan endpoint internal `POST /api/v1/samples/:id/convert-to-order` dengan otorisasi staf internal (`owner_executive`, `production_head`).

### B. Modul Client Portal (`internal/modules/portal`)
- **[`model.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/portal/model.go):**
  - Menambahkan struct `ClientPortalSampleSummary`, `ClientPortalSampleDetail`, `ClientPortalApproveSamplePayload`, `ClientPortalRequestRevisionPayload`, dan sentinel error `ErrSampleNotFound`.
- **[`repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/portal/repository.go) & [`postgres_repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/portal/postgres_repository.go):**
  - `ListClientSamples(ctx, clientID, status)`: Mengambil daftar sample terfilter strictly per client tenant.
  - `GetClientSampleDetail(ctx, clientID, sampleID)`: Mengambil detail lengkap sample beserta seluruh riwayat versi dan breakdown time study komponen (anti-IDOR).
  - `ApproveSample(ctx, clientID, sampleID, feedbackNotes)`: Mengunci versi aktif menjadi `APPROVED`, mencatat tanggal persetujuan, dan menyinkronkan nilai SAM ke masterdata `components`.
  - `RequestSampleRevision(ctx, clientID, sampleID, revisionNotes)`: Mengubah status versi menjadi `REVISED` dan mencatat catatan revisi klien.
- **[`service.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/portal/service.go) & [`handler.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/portal/handler.go):**
  - Mengintegrasikan layanan notifikasi otomatis (`notification.Service`): Memicu notifikasi in-app `SAMPLE_APPROVED` dan `SAMPLE_REVISION_REQUESTED`.
  - Mendaftarkan endpoint portal:
    - `GET /api/v1/portal/samples`
    - `GET /api/v1/portal/samples/:id`
    - `POST /api/v1/portal/samples/:id/approve`
    - `POST /api/v1/portal/samples/:id/request-revision`

### C. Modul Order (`internal/modules/order`)
- **[`model.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/order/model.go) & [`postgres_repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/order/postgres_repository.go):**
  - Menambahkan field `SampleVersionID`, `UnitPrice`, dan `TotalAmount` pada `InternalOrderSummary` dan `InternalOrderDetail` sehingga order hasil konversi sample menampilkan informasi komersial dan referensi versi secara transparan di dashboard pabrik.

### D. Dependency Wiring (`internal/app`)
- **[`app.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/app/app.go):**
  - Menghubungkan `notificationSvc` ke dalam `portal.NewService(portalRepo, notificationSvc)`.

---

## 4. Daftar & Spesifikasi Endpoint Hasil Tahap 6.2

| No | Method | Endpoint | Akses Peran | Deskripsi Bisnis |
|---|---|---|---|---|
| 1 | `GET` | `/api/v1/portal/samples?status=ALL` | `client` | Klien melihat daftar seluruh pengajuan sample miliknya di portal |
| 2 | `GET` | `/api/v1/portal/samples/:id` | `client` | Klien melihat detail spesifikasi sample, breakdown SAM, dan harga penawaran CMT |
| 3 | `POST` | `/api/v1/portal/samples/:id/request-revision` | `client` | Klien meminta perbaikan pola/aksesoris sample (status beralih ke `REVISED`) |
| 4 | `POST` | `/api/v1/samples/:id/versions` | `internal` | Tim pabrik menerbitkan revisi versi 2 (`v2`) |
| 5 | `POST` | `/api/v1/samples/:id/versions/:ver/time-studies` | `internal` | Observasi ulang stopwatch SAM pada komponen revisi versi 2 |
| 6 | `PUT` | `/api/v1/samples/:id/versions/:ver/commercial-rate` | `internal` | Penetapan tarif maklon CMT per potong ke klien (Commercial Rate Card) |
| 7 | `POST` | `/api/v1/samples/:id/submit-review` | `internal` | Mengajukan versi 2 ke klien dan memicu notifikasi `SAMPLE_READY_FOR_REVIEW` |
| 8 | `POST` | `/api/v1/portal/samples/:id/approve` | `client` | Klien menyetujui versi 2 beserta harga Rp 45.000 (status beralih ke `APPROVED`) |
| 9 | `POST` | `/api/v1/samples/:id/convert-to-order` | `internal` | Konversi sample approved menjadi Order Produksi Massal 2.000 pcs (Skenario Uji 4) |
| 10 | `GET` | `/api/v1/orders/:id` | `internal` | Pengawasan order massal pabrik dengan bukti binding spesifikasi versi 2 & total nilai kontrak |

---

## 5. Bukti Pengujian Otomatis

### A. Unit Tests Seluruh Modul
- **`internal/modules/sample`:** 6/6 tests PASS (`TestSampleService_CreateSampleRequest_Success`, `TestSampleService_Versioning_v1_to_v2`, `TestSampleService_RecordTimeStudy_SAM_Calculation`, `TestSampleService_SetCommercialRate`, `TestSampleService_SubmitReview`, `TestSampleService_ConvertToOrder`).
- **`internal/modules/portal`:** 4/4 tests PASS (`TestPortalService_MultiTenantValidation`, `TestPortalService_OrderDrillDownAndStocks`, `TestPortal_ModelStructs`, `TestPortalService_SampleApprovalAndRevision`).
- **`internal/modules/order`:** 1/1 test PASS (`TestOrderService_CompleteOrderFlow`).

### B. Live Database Acceptance Test Skenario Uji 4 (`TestApp_SampleScenario4LiveDB`)
Eksekusi 13 langkah pengujian berurutan pada live database PostgreSQL (`localhost:5433/lukla_db`):
```text
=== RUN   TestApp_SampleScenario4LiveDB
    app_test.go:4984: Langkah 1 PASS: Sample SMP-202610-009 dibuat (v1 id: 45bd26ed-2028-4087-8588-ddf02b640670)
    app_test.go:5022: Langkah 2 PASS: Time study v1 dicatat (Total SAM v1 = 14.95)
    app_test.go:5039: Langkah 3 PASS: Sample v1 diajukan ke review klien
    app_test.go:5059: Langkah 4 PASS: Klien Snøhund berhasil meninjau sample v1 di portal
    app_test.go:5093: Langkah 5 PASS: Permintaan revisi klien terkirim, status REVISED, notifikasi tercatat
    app_test.go:5123: Langkah 6 PASS: Versi 2 diterbitkan (v2 id: 0e265c14-ec85-4a9e-861c-125d92c1504d)
    app_test.go:5150: Langkah 7 PASS: Time study v2 diperbarui. Total SAM v2 = 13.80
    app_test.go:5169: Langkah 8 PASS: Commercial Rate Card ditetapkan Rp 45.000 pada v2
    app_test.go:5186: Langkah 9 PASS: Sample v2 diajukan ke review klien
    app_test.go:5215: Langkah 10 PASS: Klien meninjau v2 di portal, harga Rp 45.000 dan SAM 13.80 terverifikasi
    app_test.go:5256: Langkah 11 PASS: Sample v2 disetujui klien, notifikasi SAMPLE_APPROVED dikirim, masterdata SAM disinkronkan
    app_test.go:5319: Skenario 4 Proof: Order massal terbukti mengikat spesifikasi versi v2 (0e265c14-ec85-4a9e-861c-125d92c1504d), BUKAN v1 (45bd26ed-2028-4087-8588-ddf02b640670)
    app_test.go:5332: Skenario 4 Proof: Unit Price Rp 45.000 x 2.000 pcs = Rp 90.000.000 akurat 100%
    app_test.go:5340: Skenario 4 Proof: Sample request terkunci dengan converted_order_id=2d225463-b0ca-4f7f-ac67-ab9e61c3d1f0 dan status=CONVERTED_TO_ORDER
    app_test.go:5351: Skenario 4 Proof: Upaya re-konversi diblokir secara aman (400 Bad Request)
    app_test.go:5370: Skenario 4 Proof: Multi-tenant portal isolation terverifikasi kebal terhadap IDOR lintas brand
--- PASS: TestApp_SampleScenario4LiveDB (1.69s)
PASS
```

### C. Full Test Suite Engine
`go test ./...` dijalankan pada seluruh package backend: **100% PASS** (21.97s).

---

## 6. Sinkronisasi Postman Collection

File [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) telah disinkronkan dan memuat **Folder 15: `15. Tahap 6.2: Client Approval, Commercial Rate Card & Mass Order Conversion (BPMN 03 & Skenario Uji 4)`** dengan 11 request lengkap beserta pengujian JavaScript otomatis (`pm.test`) dan **respons nyata (*real response*)** dari hasil eksekusi pengujian:
1. `15.1 Klien Tinjau Daftar Sample di Portal`: 200 OK.
2. `15.2 Klien Tinjau Detail Sample & Riwayat Versi (v1)`: 200 OK.
3. `15.3 Klien Ajukan Permintaan Revisi Spesifikasi (v1 -> Revisi)`: 200 OK.
4. `15.4 Pabrik Terbitkan Versi Revisi 2 (v2)`: 201 Created.
5. `15.5 Update Time Study Observasi Komponen pada Versi 2`: 201 Created.
6. `15.6 Tetapkan Commercial Rate Card ke Klien pada Versi 2`: 200 OK (`proposedUnitPrice: 45000`).
7. `15.7 Pabrik Ajukan Versi 2 ke Klien untuk Persetujuan`: 200 OK.
8. `15.8 Klien Tinjau Versi 2 & Harga Commercial Rate Card di Portal`: 200 OK.
9. `15.9 Klien Menyetujui Versi Sample & Penawaran Harga (Digital Approval)`: 200 OK (`status: APPROVED`).
10. `15.10 Konversi Sample Approved Menjadi Order Produksi Massal (Skenario Uji 4)`: 201 Created (`approvedVersion: 2`, `unitPrice: 45000`, `totalAmount: 90000000`).
11. `15.11 Verifikasi Detail Order Massal Mengikat Versi 2 & Commercial Rate`: 200 OK.

