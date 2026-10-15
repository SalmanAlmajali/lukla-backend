# Laporan Pengerjaan Tahap 6.1: Sample Lifecycle, Versioning Engine, & Time Study Tracker (BPMN 03)

**Tanggal Penyelesaian:** 15 Oktober 2026  
**Status:** Selesai (100% PASS - Unit Tests, Acceptance Tests Live DB, & Full Test Suite)  
**Dokumen Referensi Perencanaan:** [analisis-dan-rencana-tahap-6.md](./analisis-dan-rencana-tahap-6.md)  
**Spesifikasi OpenAPI:** [openapi-phase6-sample-time-study.yaml](../openapi-phase6-sample-time-study.yaml)  
**Database Migration:** Versi 32 (`000032_create_samples_and_time_study_tables.up.sql`)  
**Modul Baru Engine:** `internal/modules/sample`  
**Modul yang Disempurnakan:** `internal/modules/notification`, `internal/app`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Berdasarkan diagram proses bisnis **BPMN 03 (Sample ke Order Massal)**, sebelum brand klien menerbitkan Purchase Order (PO) bernilai ratusan juta rupiah untuk ribuan potong pakaian, pabrik maklon dan klien harus melalui fase pembuatan prototipe fisik (*sample development*).

Tahap 6.1 mengimplementasikan fondasi teknis siklus sample ini:
1. **Inisiasi Permintaan Sample Baru:**
   - Pembuatan permintaan sample (`sample_requests`) untuk kombinasi klien dan artikel tertentu.
   - Sistem secara otomatis menginisialisasi versi awal `v1` berstatus `DRAFT`.
   - Kode sample unik dihasilkan secara sekuensial dengan pola `SMP-YYYYMM-XXX` (misal: `SMP-202610-001`).
2. **Versioning Engine Bertingkat & Immutable Audit Trail:**
   - Saat klien meminta perbaikan pada sample fisik (misal: saku dinaikkan, zipper diganti, ukuran lingkar dada disesuaikan), sistem tidak mengizinkan penimpaan data secara diam-diam.
   - Melalui endpoint `POST /samples/:id/versions`:
     - Versi sebelumnya (`v1`) otomatis dikunci (*immutable lock*) dengan status `REVISED` dan tidak dapat diubah lagi.
     - Versi baru (`v2`) diterbitkan dengan status `DRAFT` dan mewarisi catatan revisi dari klien.
     - Opsi `copyTimeStudiesFromPrevious: true` secara otomatis menduplikasi seluruh catatan pengukuran komponen dari versi lama ke versi baru sehingga tim IE (*Industrial Engineering*) hanya perlu menyesuaikan komponen yang berubah.
3. **Time Study Tracker (Stopwatch SAM):**
   - Mengakomodasi observasi waktu kerja operator jahit/potong menggunakan stopwatch per komponen artikel.
   - Formula Standard Allowed Minute (SAM) industri garmen internasional:
     $$\text{Calculated SAM} = \frac{\text{Observed Seconds} \times (1 + \frac{\text{Allowance \%}}{100})}{60.0}$$
     $$\text{Target Per Hour} = \lfloor \frac{60.0}{\text{Calculated SAM}} \rfloor$$
   - Total SAM per versi dihitung dan diakumulasikan secara otomatis dari seluruh komponen.
4. **Penetapan Commercial Rate Card:**
   - Dari hasil kalkulasi total SAM yang didapat, manajemen/owner dapat menetapkan tarif maklon per potong baju ke klien (`proposedUnitPrice`, `currency`, `priceNotes`) melalui `PUT /samples/:id/versions/:ver/commercial-rate`.
5. **Pengajuan ke Klien & Integrasi Notifikasi In-App:**
   - Ketika pembuatan sample fisik di pabrik selesai, staf mengeksekusi `POST /samples/:id/submit-review`.
   - Status sample dan versi aktif bertransisi menjadi `IN_REVIEW`.
   - Sistem otomatis memicu notifikasi in-app ber-tipe `SAMPLE_READY_FOR_REVIEW` ke portal akun klien terkait.

---

## 2. Perubahan Skema Database (Migration 32)

Diterapkan file migrasi PostgreSQL live:
- `migrations/db/migrations/000032_create_samples_and_time_study_tables.up.sql`
- `migrations/db/migrations/000032_create_samples_and_time_study_tables.down.sql`

### Rincian Perubahan Skema:
```sql
-- 1. Tabel Permintaan Sample (sample_requests)
CREATE TABLE IF NOT EXISTS sample_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_code VARCHAR(50) NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE RESTRICT,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    target_date DATE,
    initial_notes TEXT,
    converted_order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tabel Versi Sample (sample_versions) - Versioning Engine
CREATE TABLE IF NOT EXISTS sample_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_request_id UUID NOT NULL REFERENCES sample_requests(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    version_label VARCHAR(20) NOT NULL, -- 'v1', 'v2', dst
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- DRAFT, IN_DEVELOPMENT, IN_REVIEW, REVISED, APPROVED, REJECTED
    revision_notes TEXT,
    client_feedback TEXT,
    total_sam NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    proposed_unit_price NUMERIC(15,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    price_notes TEXT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sample_versions_request_num UNIQUE (sample_request_id, version_number)
);

-- 3. Tabel Observasi Waktu Kerja Stopwatch (sample_time_studies) per Komponen
CREATE TABLE IF NOT EXISTS sample_time_studies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_version_id UUID NOT NULL REFERENCES sample_versions(id) ON DELETE CASCADE,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    observed_seconds NUMERIC(10,2) NOT NULL CHECK (observed_seconds > 0),
    allowance_percentage NUMERIC(5,2) NOT NULL DEFAULT 15.00 CHECK (allowance_percentage >= 0),
    calculated_sam NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    target_per_hour INT NOT NULL DEFAULT 0,
    operator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sample_time_studies_version_comp UNIQUE (sample_version_id, component_id)
);

-- 4. Kolom Tambahan pada Tabel orders (Persiapan Konversi Tahap 6.2)
ALTER TABLE orders ADD COLUMN IF NOT EXISTS sample_version_id UUID REFERENCES sample_versions(id) ON DELETE SET NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS unit_price NUMERIC(15,2);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_amount NUMERIC(15,2);
```

---

## 3. Arsitektur & Komponen Kode

### 1. Modul Baru `internal/modules/sample`
* **`model.go`**: Mendefinisikan model domain `SampleRequestSummary`, `SampleRequestDetail`, `SampleVersionDetail`, `SampleTimeStudyItem`, dan struct payload request/response.
* **`repository.go`**: Interface repository untuk isolasi persistensi data sample, versioning, dan stopwatch time study.
* **`postgres_repository.go`**: Implementasi database PostgreSQL dengan transaction locking (`SELECT ... FOR UPDATE`), generator kode sample sekuensial, auto-copy components antar versi, dan kalkulasi otomatis akumulasi `total_sam`.
* **`service.go`**: Logika bisnis validasi transisi status, kalkulasi SAM & target per jam, dan event dispatcher notifikasi in-app ke akun klien.
* **`handler.go`**: HTTP controller Gin untuk 8 endpoint internal manajemen sample.
* **`service_test.go`**: Unit test independen (5 skenario pengujian dengan mock repository).

### 2. Modul `internal/modules/notification`
* Penambahan konstanta event type:
  * `EventTypeSampleReadyForReview = "SAMPLE_READY_FOR_REVIEW"`
  * `EventTypeSampleApproved = "SAMPLE_APPROVED"`
  * `EventTypeSampleRevisionRequested = "SAMPLE_REVISION_REQUESTED"`
* Penguatan switch-case validation pada `SendNotification`.

### 3. Modul `internal/app`
* Wiring `sampleRepo`, `sampleSvc`, dan `sampleHandler` ke router Gin engine dengan middleware otentikasi JWT dan autorisasi peran internal (`owner_executive`, `production_head`, `supervisor`).

---

## 4. Spesifikasi Endpoint yang Diimplementasikan

| Method | Endpoint | Peran yang Diizinkan | Deskripsi Fungsi |
|---|---|---|---|
| `POST` | `/api/v1/samples` | Internal Staf | Membuat permintaan sample baru & otomatis inisialisasi versi 1 (`v1`). |
| `GET` | `/api/v1/samples` | Internal Staf | Mengambil daftar seluruh sample dengan filter clientId, status, search, & pagination. |
| `GET` | `/api/v1/samples/:id` | Internal Staf | Mengambil detail lengkap sample beserta riwayat seluruh versi dan time study. |
| `POST` | `/api/v1/samples/:id/versions` | Internal Staf | Menerbitkan versi revisi baru (`v1` $\to$ `v2`) dan mengunci versi lama (`REVISED`). |
| `GET` | `/api/v1/samples/:id/versions/:ver/time-studies` | Internal Staf | Rekapitulasi hasil observasi time study per komponen pada versi tertentu. |
| `POST` | `/api/v1/samples/:id/versions/:ver/time-studies` | Internal Staf | Mencatat observasi stopwatch (detik teramati, allowance %, kalkulasi SAM). |
| `PUT` | `/api/v1/samples/:id/versions/:ver/commercial-rate` | Internal Staf | Menetapkan / memperbarui penawaran harga maklon per potong ke klien. |
| `POST` | `/api/v1/samples/:id/submit-review` | Internal Staf | Mengajukan sample fisik ke klien (`IN_REVIEW`) dan memicu notifikasi in-app. |

---

## 5. Bukti Pengujian Otomatis

### A. Unit Tests Modul Sample (`internal/modules/sample`)
```bash
=== RUN   TestSampleService_CreateSampleRequest_Success
--- PASS: TestSampleService_CreateSampleRequest_Success (0.00s)
=== RUN   TestSampleService_Versioning_v1_to_v2
--- PASS: TestSampleService_Versioning_v1_to_v2 (0.00s)
=== RUN   TestSampleService_RecordTimeStudy_SAM_Calculation
--- PASS: TestSampleService_RecordTimeStudy_SAM_Calculation (0.00s)
=== RUN   TestSampleService_SetCommercialRate
--- PASS: TestSampleService_SetCommercialRate (0.00s)
=== RUN   TestSampleService_SubmitReview
--- PASS: TestSampleService_SubmitReview (0.00s)
PASS
ok  	github.com/lukla/operations-engine/internal/modules/sample	2.681s
```

### B. Acceptance Test Live Database (`TestApp_SampleLifecycleAndTimeStudyLiveDB`)
```bash
=== RUN   TestApp_SampleLifecycleAndTimeStudyLiveDB
    app_test.go:4631: Step 1 PASS: Sample SMP-202610-002 created with initial version v1
    app_test.go:4655: Step 2 PASS: Listed samples for client f310f933-4b79-40c4-849d-f0334716d43c, found 1
    app_test.go:4690: Step 3 PASS: Recorded Time Study 1: SAM=9.20, Target/Hour=6
    app_test.go:4735: Step 4 PASS: Total SAM v1 verified: 14.95 (2 components)
    app_test.go:4777: Step 5 PASS: Version 2 published. Version 1 is locked as REVISED
    app_test.go:4805: Step 6 PASS: Updated Time Study on v2. New Total SAM: 13.80
    app_test.go:4835: Step 7 PASS: Commercial Rate Card set to Rp 45.000 for version 2
    app_test.go:4870: Step 8 PASS: Sample submitted to review. In-app notification created for client (count: 1)
    app_test.go:4900: Step 9 PASS: RBAC, locking enforcement, and unauthorized guards verified
--- PASS: TestApp_SampleLifecycleAndTimeStudyLiveDB (1.04s)
PASS
```

### C. Full Test Suite Regression (`go test ./...`)
Seluruh 19 package sistem lulus uji tanpa kegagalan:
```bash
ok  	github.com/lukla/operations-engine/internal/app	23.184s
ok  	github.com/lukla/operations-engine/internal/modules/identity	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/inventory	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/masterdata	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/notification	4.582s
ok  	github.com/lukla/operations-engine/internal/modules/order	3.031s
ok  	github.com/lukla/operations-engine/internal/modules/payroll	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/portal	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/production	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/qcbonus	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/quality	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/ratecard	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/sample	2.101s
PASS
```

---

## 6. Sinkronisasi Postman Collection

Berkas Postman Collection `Lukla_Articles_and_Components.postman_collection.json` telah diperbarui dengan menyematkan **Folder 14: Tahap 6.1: Sample & Time Study Engine (BPMN 03)** yang mencakup 10 request uji dengan scripts `pm.test` otomatis:
1. `POST {{base_url}}/samples` - Buat Permintaan Sample Baru (Inisialisasi v1)
2. `GET {{base_url}}/samples?status=ALL&limit=10` - Daftar Seluruh Permintaan Sample Pabrik
3. `GET {{base_url}}/samples/{{sample_id}}` - Detail Lengkap Permintaan Sample & Riwayat Versi
4. `POST {{base_url}}/samples/{{sample_id}}/versions/1/time-studies` - Input Observasi Time Study Komponen 1 (SAM)
5. `POST {{base_url}}/samples/{{sample_id}}/versions/1/time-studies` - Input Observasi Time Study Komponen 2 (SAM)
6. `GET {{base_url}}/samples/{{sample_id}}/versions/1/time-studies` - Rekapitulasi Time Study Versi 1
7. `POST {{base_url}}/samples/{{sample_id}}/versions` - Terbitkan Versi Revisi Baru (v1 -> v2 & Locking v1)
8. `POST {{base_url}}/samples/{{sample_id}}/versions/2/time-studies` - Update Observasi Time Study pada Versi 2
9. `PUT {{base_url}}/samples/{{sample_id}}/versions/2/commercial-rate` - Tetapkan Commercial Rate Card (Harga Jasa Maklon ke Klien)
10. `POST {{base_url}}/samples/{{sample_id}}/submit-review` - Ajukan Sample ke Klien (Submit to Review & Picu Notifikasi)

