# Laporan Pengerjaan Tahap 4.1: Quality Control Engine, Defect Tracking, & Verifikasi Upah Penjahit (BPMN 06)

**Tanggal Penyelesaian:** 15 Oktober 2026  
**Status:** Selesai (100% Terverifikasi Otomatis pada Live Database)  
**Database Migration:** `000029_create_qc_inspection_tables.up.sql` (DB Version: 29)  
**Target Modul:** `internal/modules/quality`, `internal/app`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Tahap 4.1 merupakan implementasi dari alur pemeriksaan mutu dan pelacakan cacat garmen (*Quality Control & Defect Tracking*) sesuai proses **BPMN 06** dan ketentuan spesifikasi kebutuhan sistem Lukla Operations Engine:

### Aturan Bisnis Utama:
1. **Antrean Produk Assembly (`WAITING_QC` / `MENUNGGU_QC`):**
   - Mengambil unit garmen yang telah selesai dirakit dari lantai jahit (hasil BPMN 05) untuk diperiksa secara fisik oleh petugas QC Inspector.
2. **Tiga Status Hasil Pemeriksaan Baku:**
   - **`LOLOS` (Passed / Barang Jadi):** Produk memenuhi seluruh standar mutu, otomatis dialokasikan menjadi basis akumulasi pool bonus QC (BPMN 08) serta memperbarui penghitung progres penyelesaian order.
   - **`REPAIR` (Perbaikan / Rework):** Produk memiliki cacat yang dapat diperbaiki (misal: jahitan loncat atau kerutan). Wajib mencatat rincian defect & komponen terkait, lalu dikembalikan ke penjahit untuk diperbaiki.
   - **`REJECT` (Cacat Permanen / Afkir):** Produk cacat parah yang tidak dapat diperbaiki (misal: panel kain terpotong terlalu pendek atau cacat bahan parah).
3. **Integritas Upah Penjahit (BPMN 06 / Aturan Klien Kritis):**
   - **Produk REJECT tidak memotong upah borongan penjahit.**
   - Upah penjahit murni dihitung dari kuantitas kerja yang telah diverifikasi oleh mandor di lantai produksi (`production_outputs`). Hasil penolakan QC tidak boleh mengurangi penerimaan penjahit.
4. **Analitik Mutu & Rekapitulasi per Order:**
   - Menghitung persentase kelulusan (*Pass Rate*), total produk lolos, repair, dan reject, serta menyajikan daftar kategori defect yang paling sering muncul (*Top Defects*) untuk evaluasi lini jahit.

---

## 2. Perubahan Skema Database

File migrasi yang diterapkan ke live database PostgreSQL (`lukla_db`):
- `migrations/db/migrations/000029_create_qc_inspection_tables.up.sql`
- `migrations/db/migrations/000029_create_qc_inspection_tables.down.sql`

### Rincian Tabel Baru:
1. **`qc_defect_categories` (Master Katalog Defect):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `code` (VARCHAR(50), NOT NULL, UNIQUE, e.g. `DEF-SEW-01`)
   - `name` (VARCHAR(100), NOT NULL)
   - `severity` (VARCHAR(20), CHECK `severity IN ('MINOR', 'MAJOR', 'CRITICAL')`)
   - `description` (TEXT)
   - `is_active` (BOOLEAN, DEFAULT TRUE)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
   - *Seeded Kategori Bawaan:*
     - `DEF-SEW-01`: Jahitan Loncat / Lepas (`MAJOR`)
     - `DEF-WRI-01`: Kerutan / Puckering (`MINOR`)
     - `DEF-STN-01`: Noda Kain / Minyak Mesin (`MAJOR`)
     - `DEF-SIZ-01`: Ukuran Meleset / Measurement Off (`CRITICAL`)
     - `DEF-ACC-01`: Aksesoris Rusak / Cacat (`MAJOR`)

2. **`qc_inspections` (Transaksi Pemeriksaan QC):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `inspection_number` (VARCHAR(50), NOT NULL, UNIQUE, format: `QC-YYYYMMDD-XXXX`)
   - `assembly_id` (UUID, NOT NULL, FK `assemblies(id)`)
   - `order_id` (UUID, NOT NULL, FK `orders(id)`)
   - `inspector_id` (UUID, NOT NULL, FK `users(id)`)
   - `inspection_date` (DATE, NOT NULL, DEFAULT CURRENT_DATE)
   - `total_checked` (INT, CHECK `total_checked > 0`)
   - `passed_quantity` (INT, CHECK `passed_quantity >= 0`)
   - `repair_quantity` (INT, CHECK `repair_quantity >= 0`)
   - `reject_quantity` (INT, CHECK `reject_quantity >= 0`)
   - `notes` (TEXT)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
   - *Constraint:* `total_checked = passed_quantity + repair_quantity + reject_quantity`

3. **`qc_inspection_defects` (Detail Temuan Cacat):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `inspection_id` (UUID, NOT NULL, FK `qc_inspections(id)` ON DELETE CASCADE)
   - `defect_category_id` (UUID, NOT NULL, FK `qc_defect_categories(id)`)
   - `component_id` (UUID, NULLABLE, FK `components(id)` ON DELETE SET NULL)
   - `quantity` (INT, CHECK `quantity > 0`)
   - `disposition` (VARCHAR(20), CHECK `disposition IN ('REPAIR', 'REJECT')`)
   - `notes` (TEXT)
   - `created_at` (TIMESTAMPTZ)

---

## 3. Implementasi Modul Go (`internal/modules/quality`)

### A. Komponen Kode:
1. **Model (`internal/modules/quality/model.go`):**
   - Struct data: `DefectCategory`, `CreateDefectCategoryRequest`, `QCQueueItem`, `ListQCQueueFilter`, `CreateQCInspectionRequest`, `CreateQCInspectionDefect`, `QCInspectionSummary`, `ListQCInspectionsFilter`, `QCInspectionDetail`, `QCInspectionDefect`, `QCOrderSummary`, `TopDefectItem`.
   - Error sentinel: `ErrDefectCategoryNotFound`, `ErrDefectCategoryCodeExists`, `ErrInspectionNotFound`, `ErrAssemblyNotFound`, `ErrOrderNotFound`, `ErrInvalidQuantity`, `ErrDefectDetailsRequired`, `ErrDefectQuantityMismatch`, `ErrForbiddenQuality`.
2. **Repository (`internal/modules/quality/repository.go` & `postgres_repository.go`):**
   - `ListDefectCategories`, `CreateDefectCategory`, `FindDefectCategoryByID`.
   - `ListQCQueue`: Query antrean perakitan yang berstatus `WAITING_QC` atau `REPAIR` dengan join detail order, client, artikel, dan alokasi line.
   - `CreateInspection`: Eksekusi transaksional atomik:
     - Validasi lock `FOR UPDATE` baris assembly.
     - Penomoran otomatis format `QC-YYYYMMDD-XXXX`.
     - Insert header transaksi dan detail item cacat.
     - Transisi status assembly (`PASSED`, `REPAIR`, `REJECTED`, atau `PARTIALLY_PASSED`).
     - Deteksi penyelesaian order jika total pcs lolos telah mencapai target pesanan.
   - `ListInspections` & `FindInspectionByID`: Query ringkasan dan detail riwayat inspeksi beserta daftar defect.
   - `GetOrderQualitySummary`: Kalkulasi agregasi kuantitas lolos, repair, reject, persentase *Pass Rate*, dan peringkat *Top Defects*.
3. **Service Layer (`internal/modules/quality/service.go`):**
   - Validasi ketat integritas jumlah repair & reject terhadap rincian detail defect (`repairCount == req.RepairQuantity` dan `rejectCount == req.RejectQuantity`).
   - Penegakan otorisasi role (`qc_inspector`, `production_head`, `owner_executive`).
4. **Handler Layer (`internal/modules/quality/handler.go`):**
   - Registrasi endpoint router Gin:
     - `GET /api/v1/qc/defect-categories`
     - `POST /api/v1/qc/defect-categories`
     - `GET /api/v1/qc/queue`
     - `GET /api/v1/qc/inspections`
     - `POST /api/v1/qc/inspections`
     - `GET /api/v1/qc/inspections/:id`
     - `GET /api/v1/qc/orders/:orderId/summary`

---

## 4. Hasil Pengujian Otomatis

### A. Unit Testing (`internal/modules/quality/quality_test.go`)
- `TestQualityService_CreateDefectCategory`: Lulus verifikasi RBAC penjahit dilarang menambah kategori defect, sukses untuk QC Inspector, dan penolakan duplikasi kode.
- `TestQualityService_CreateInspection_Validation`: Lulus verifikasi penolakan kuantitas 0, penolakan jika repair > 0 tanpa rincian defect, penolakan mismatch selisih kuantitas defect, dan sukses inspeksi valid.

### B. Acceptance Test Live Database (`internal/app/app_test.go`)
Fungsi uji `TestApp_QCInspectionLiveDB` memverifikasi seluruh alur operasional pada database riil:
1. **Master Kategori Defect:** Mengambil 5 kategori bawaan dan menambahkan kategori baru `DEF-LBL-01`.
2. **Antrean QC:** Mengambil batch assembly jahit dengan status `WAITING_QC`.
3. **Pemeriksaan Lolos & Repair:** Memeriksa batch 50 pcs (48 pcs Lolos, 2 pcs Repair jahitan loncat). Status assembly otomatis beralih menjadi `PARTIALLY_PASSED`.
4. **Pemeriksaan Reject & Pembuktian Upah Penjahit:**
   - Memeriksa batch kedua 10 pcs (8 pcs Lolos, 2 pcs Reject cacat panel).
   - **Verifikasi Integritas Upah Penjahit:** Data output penjahit di tabel `production_outputs` dipastikan tetap utuh (1.869 pcs), membuktikan bahwa reject QC sama sekali tidak memotong upah borongan penjahit.
5. **Analitik Kualitas Order:** Memverifikasi kalkulasi pass rate (93.33%), total checked (60 pcs), total passed (56 pcs), total repair (2 pcs), total reject (2 pcs), serta ranking top defect.

---

## 5. Sinkronisasi Postman Collection

Berkas Postman Collection [`Lukla_Articles_and_Components.postman_collection.json`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/Lukla_Articles_and_Components.postman_collection.json) telah dilengkapi dengan folder baru:
📁 **`10. Tahap 4: Quality Control & Defect Tracking (BPMN 06)`**

Berisi 9 request uji siap pakai dengan contoh respons riil dari engine:
1. `1. Login QC Inspector (Siti Handayani)` $\to$ `200 OK`
2. `2. GET /qc/defect-categories - Master Kategori Defect` $\to$ `200 OK`
3. `3. POST /qc/defect-categories - Tambah Kategori Defect Baru` $\to$ `201 Created`
4. `4. GET /qc/queue - Antrean Assembly Siap QC` $\to$ `200 OK`
5. `5. POST /qc/inspections - Submit Hasil Periksa QC (48 Lolos, 2 Repair)` $\to$ `201 Created`
6. `6. GET /qc/inspections/:id - Detail Hasil Periksa & Log Defect` $\to$ `200 OK`
7. `7. POST /qc/inspections - Submit Hasil Periksa QC dengan Reject (8 Lolos, 2 Reject)` $\to$ `201 Created`
8. `8. GET /qc/orders/:orderId/summary - Analitik Kualitas & Pass Rate per Order` $\to$ `200 OK`
9. `9. GET /qc/inspections - Riwayat Inspeksi QC` $\to$ `200 OK`

