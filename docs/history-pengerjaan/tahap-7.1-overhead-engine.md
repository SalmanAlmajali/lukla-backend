# Laporan Pengerjaan Tahap 7.1: Factory Overhead Engine, Monthly Cost Allocation & COGS Report (BPMN 09 & BPMN 01)

**Tanggal:** 15 Oktober 2026  
**Status:** SELESAI 100% (Terverifikasi di Live Database)  
**Referensi:**
- Roadmap: `urutan pengerjaan.txt` (Tahap 7: BPMN 09, HPP Order BPMN 01, Skenario Uji End-to-End)
- Diagram Proses Bisnis: `Lukla_BPMN_v2.drawio` (Diagram 9: Alokasi Biaya Overhead Bulanan Pabrik & Diagram 1: Analisis HPP Order)
- Rencana Kerja: [analisis-dan-rencana-tahap-7.md](./analisis-dan-rencana-tahap-7.md)
- Spesifikasi Kontrak API: [openapi-phase7-overhead-e2e.yaml](../openapi-phase7-overhead-e2e.yaml)
- Koleksi Postman: [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) (Folder 16)

---

## 1. Latar Belakang & Kebutuhan Bisnis

Tahap 7.1 mengimplementasikan mesin akuntansi biaya overhead pabrik (*Factory Overhead Engine*) sesuai standar BPMN 09 serta modul pelaporan HPP (*Cost of Goods Sold* / COGS) komprehensif per order sesuai standar BPMN 01:

1. **Pentingnya Biaya Overhead Pabrik (BPMN 09):**
   - Biaya langsung seperti bahan baku (*Direct Materials*) dan upah penjahit (*Direct Labor*) dapat ditelusuri langsung ke tiap potong pakaian. Namun, operasional pabrik garmen memerlukan biaya tidak langsung yang signifikan: tagihan listrik mesin jahit & kompresor, sewa gedung fasilitas workshop, gaji mandor & staf gudang, penyusutan mesin jahit (*depreciation*), biaya oli/perawatan jarum, dan bahan pembantu/ATK packing.
   - Tanpa alokasi overhead yang sistematis, manajemen pabrik Lukla tidak dapat mengetahui profitabilitas riil dari setiap order maklon yang dikerjakan.
2. **Metode Alokasi Biaya: Tarif Rata (*Flat Rate*) vs Terbobot (*Weighted Rate*):**
   - **Flat Rate Allocation:**
     $$\text{Tarif Flat Per Unit} = \frac{\text{Total Biaya Overhead Riil}}{\text{Total Unit Lolos QC}}$$
   - **Weighted Category Allocation (Garment vs Bag):**
     Produksi tas ransel / tas gunung (*BAG*) membutuhkan konsumsi listrik mesin jarum ganda (*walking foot/heavy-duty*), benang tebal, dan waktu perakitan lebih intensif dibandingkan kemeja/jaket biasa (*GARMENT*).
     Sistem menerapkan faktor bobot yang dapat dikonfigurasi:
     - Garment Weight: $1.00$
     - Bag Weight: $1.25$
     $$\text{Total Weighted Units} = (Q_{\text{Garment}} \times W_{\text{Garment}}) + (Q_{\text{Bag}} \times W_{\text{Bag}})$$
     $$\text{Tarif Garment Per Unit} = \left(\frac{\text{Total Biaya Overhead}}{\text{Total Weighted Units}}\right) \times W_{\text{Garment}}$$
     $$\text{Tarif Bag Per Unit} = \left(\frac{\text{Total Biaya Overhead}}{\text{Total Weighted Units}}\right) \times W_{\text{Bag}}$$
3. **Siklus Hidup & Proteksi Periode (*Immutability Guard*):**
   - Periode overhead bulanan bergerak melalui 3 status: `OPEN` $\rightarrow$ `CALCULATED` $\rightarrow$ `LOCKED`.
   - Ketika periode berstatus `LOCKED`, seluruh catatan pengeluaran dibekukan permanen. Upaya penambahan, penyuntingan, atau kalkulasi ulang otomatis ditolak dengan pesan `HTTP 400 Bad Request / 409 Conflict` demi kepatuhan audit akuntansi.
4. **Analisis HPP / COGS Komprehensif (BPMN 01):**
   - Mengintegrasikan tiga pilar biaya manufaktur:
     1. **Direct Labor:** Upah borongan per komponen dari master *rate cards*.
     2. **QC Passed Bonus:** Bonus lolos inspeksi kualitas per pcs (Rp 500/pcs dari modul `qcbonus`).
     3. **Factory Overhead Allocation:** Tarif serapan overhead pabrik dari periode produksi aktif.
   - Menghitung Gross Profit Margin:
     $$\text{Total HPP Per Unit} = \text{Direct Labor} + \text{QC Bonus} + \text{Overhead}$$
     $$\text{Gross Margin Per Unit} = \text{Harga Kontrak Klien} - \text{Total HPP Per Unit}$$
     $$\text{Gross Margin \%} = \left(\frac{\text{Total Gross Margin}}{\text{Total Nilai Kontrak}}\right) \times 100\%$$

---

## 2. Perubahan Skema Database (Migrasi 33)

Skema database PostgreSQL diperluas melalui file migrasi:
- [`000033_create_overhead_tables.up.sql`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/migrations/db/migrations/000033_create_overhead_tables.up.sql)
- [`000033_create_overhead_tables.down.sql`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/migrations/db/migrations/000033_create_overhead_tables.down.sql)

Tabel yang dibuat:
1. `overhead_categories`:
   - Menyimpan kategori master biaya overhead (Listrik, Sewa Bangunan, Perawatan Mesin, Depresiasi, Gaji Tidak Langsung, Bahan Pembantu, Utilitas Air, Telepon/Internet).
   - Memiliki flag `is_default` dan `is_active`.
2. `overhead_periods`:
   - Menyimpan buku alokasi bulanan (`period_code`, `period_month`, `period_year`, `start_date`, `end_date`, `status`, `total_expense_amount`, `total_produced_units`, `garment_units`, `bag_units`, `weight_garment`, `weight_bag`, `flat_rate_per_unit`, `garment_rate_per_unit`, `bag_rate_per_unit`, `locked_at`, `locked_by`).
   - Menerapkan batasan unik `UNIQUE(period_year, period_month)`.
3. `overhead_expenses`:
   - Menyimpan pos pengeluaran aktual bulanan (`overhead_period_id`, `category_id`, `expense_date`, `amount`, `invoice_reference`, `notes`).
4. `overhead_allocations`:
   - Menyimpan serapan alokasi biaya overhead ke setiap Purchase Order yang aktif/selesai pada periode terkait (`overhead_period_id`, `order_id`, `article_category`, `produced_units`, `applied_rate_per_unit`, `allocated_amount`).
   - Menerapkan batasan unik `UNIQUE(overhead_period_id, order_id)`.

---

## 3. Arsitektur & Komponen Kode yang Dibangun

### Modul Overhead (`internal/modules/overhead`)
- **[`model.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/model.go):**
  - Entitas domain: `OverheadCategory`, `OverheadPeriod`, `OverheadPeriodDetail`, `OverheadExpenseItem`, `OverheadAllocationItem`.
  - Request/Response payloads: `CreateCategoryRequest`, `OpenPeriodRequest`, `RecordExpenseRequest`, `CalculateOverheadRequest`, `OverheadCalculationResult`, `OverheadPeriodSummaryReport`, `OrderCOGSReport`.
- **[`repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/repository.go) & [`postgres_repository.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/postgres_repository.go):**
  - Query master kategori biaya overhead dan pendaftaran kategori kustom baru.
  - Pembukaan periode overhead dengan tanggal awal dan akhir bulan otomatis.
  - Pencatatan dan penghapusan biaya aktual dengan pembaruan otomatis kolom `total_expense_amount`.
  - Agregasi volume produksi lolos QC dari tabel `qc_inspections` (dengan fallback ke `orders.target_quantity`).
  - Penyimpanan alokasi overhead ke tabel `overhead_allocations` dengan pencegahan konflik (*upsert/refresh*).
  - Penguncian periode (*locking*) dengan pencatatan audit `locked_by` dan `locked_at`.
  - Kalkulasi HPP / COGS komprehensif order menggabungkan Direct Labor + QC Bonus + Overhead Rate.
- **[`service.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/service.go):**
  - Logika kalkulasi matematis tarif flat dan tarif terbobot per unit.
  - Validasi ketat terhadap periode `LOCKED`.
- **[`handler.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/handler.go):**
  - 11 endpoint Gin HTTP API dengan validasi skema request dan format response terstandarisasi.
- **[`service_test.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/modules/overhead/service_test.go):**
  - Unit test memvalidasi formula flat & weighted allocation, pembulatan 2 desimal, dan pengujian boundary.

### Wiring Aplikasi (`internal/app`)
- **[`app.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/app/app.go):**
  - Inisialisasi modul `overhead.NewPostgresRepository(db)` dan `overhead.NewService(overheadRepo)`.
  - Pendaftaran rute API di bawah `/api/v1/overhead` dan `/api/v1/reports/cogs/:orderId` dengan proteksi otentikasi internal (`owner_executive`, `finance`, `production_head`).
- **[`app_test.go`](file:///d:/Pelatihan/main%20ai/phase%203/project/lukla%20app%20project/backend/lukla-operations-engine/internal/app/app_test.go):**
  - Test skenario integrasi live database: `TestApp_OverheadLiveDB`.

---

## 4. Daftar & Spesifikasi Endpoint Hasil Tahap 7.1

| No | Method | Endpoint | Akses Peran | Deskripsi Bisnis |
|---|---|---|---|---|
| 1 | `GET` | `/api/v1/overhead/categories` | `internal` | Mengambil seluruh master kategori biaya overhead pabrik |
| 2 | `POST` | `/api/v1/overhead/categories` | `internal` | Mendaftarkan kategori biaya overhead kustom baru |
| 3 | `POST` | `/api/v1/overhead/periods` | `internal` | Membuka periode alokasi biaya overhead bulanan baru (`OPEN`) |
| 4 | `GET` | `/api/v1/overhead/periods?year=2026` | `internal` | Melihat daftar seluruh periode buku overhead yang tercatat |
| 5 | `GET` | `/api/v1/overhead/periods/:id` | `internal` | Melihat detail periode overhead, daftar pengeluaran, dan tarif |
| 6 | `POST` | `/api/v1/overhead/periods/:id/expenses` | `internal` | Mencatat beban biaya pengeluaran riil (contoh: Listrik PLN Rp 15 jt) |
| 7 | `DELETE` | `/api/v1/overhead/periods/:id/expenses/:expId` | `internal` | Menghapus pengeluaran salah catat (hanya saat belum `LOCKED`) |
| 8 | `POST` | `/api/v1/overhead/periods/:id/calculate` | `internal` | Menjalankan kalkulasi tarif flat dan terbobot (Garment 1.0 vs Tas 1.25) |
| 9 | `GET` | `/api/v1/overhead/periods/:id/summary` | `internal` | Mengambil laporan ringkasan alokasi dan persentase biaya per kategori |
| 10 | `POST` | `/api/v1/overhead/periods/:id/lock` | `internal` | Mengunci periode overhead bulanan menjadi `LOCKED` (*immutable*) |
| 11 | `GET` | `/api/v1/reports/cogs/:orderId` | `internal` | Mengambil analisis HPP komprehensif order mencakup Labor, QC Bonus & Overhead |

---

## 5. Hasil Pengujian & Bukti Eksekusi Riil

### A. Unit Test Modul Overhead
```text
=== RUN   TestOverheadService_CalculateOverhead_ZeroUnits
--- PASS: TestOverheadService_CalculateOverhead_ZeroUnits (0.00s)
=== RUN   TestOverheadService_CalculateOverhead_FlatAndWeightedSuccess
--- PASS: TestOverheadService_CalculateOverhead_FlatAndWeightedSuccess (0.00s)
=== RUN   TestOverheadService_CalculateOverhead_AllGarment
--- PASS: TestOverheadService_CalculateOverhead_AllGarment (0.00s)
PASS
ok  	github.com/lukla/operations-engine/internal/modules/overhead	3.495s
```

### B. Live Database Acceptance Test (`internal/app`)
```text
=== RUN   TestApp_OverheadLiveDB
    app_test.go:5418: Tahap 7.1 PASS: 9 Master Kategori Overhead diverifikasi
    app_test.go:5435: Tahap 7.1 PASS: Kategori overhead baru berhasil didaftarkan
    app_test.go:5470: Tahap 7.1 PASS: Periode OH-202611 berhasil dibuka
    app_test.go:5506: Tahap 7.1 PASS: Pengeluaran biaya aktual sebesar Rp 25.000.000 berhasil dicatat
    app_test.go:5537: Tahap 7.1 PASS: Kalkulasi Overhead sukses (Total: Rp 25000000.00, Flat: Rp 8333.33, Garment: Rp 8333.33, Bag: Rp 10416.67)
    app_test.go:5548: Tahap 7.1 PASS: Laporan ringkasan alokasi overhead bulanan terverifikasi
    app_test.go:5567: Tahap 7.1 PASS: Periode overhead berhasil dikunci (LOCKED)
    app_test.go:5578: Tahap 7.1 PASS: Immutability guard terbukti memblokir perubahan pada periode LOCKED
    app_test.go:5591: Tahap 7.1 PASS: Laporan COGS / HPP komprehensif berhasil diambil untuk order massal
--- PASS: TestApp_OverheadLiveDB (0.58s)
```

### C. Pengujian Regresi Keseluruhan (`go test ./...`)
Seluruh 20 package di repositori backend Lukla Operations Engine lulus 100% tanpa kendala (`ok`).

### D. Sinkronisasi Postman Collection
Folder 16 ("16. Tahap 7.1: Factory Overhead Engine & Cost Allocation (BPMN 09)") pada berkas [Lukla_Articles_and_Components.postman_collection.json](../Lukla_Articles_and_Components.postman_collection.json) telah diperbarui dengan 11 request lengkap beserta contoh respon riil dari database aktual:
- `16.1 GET Daftar Master Kategori Biaya Overhead` (200 OK)
- `16.2 POST Daftarkan Kategori Overhead Kustom` (201 Created)
- `16.3 POST Buka Periode Overhead Bulanan Baru` (201 Created)
- `16.4 GET Daftar Seluruh Periode Overhead` (200 OK)
- `16.5 GET Detail Periode Overhead Aktif` (200 OK)
- `16.6 POST Catat Biaya Pengeluaran 1 - Tagihan Listrik & Air` (201 Created)
- `16.7 POST Catat Biaya Pengeluaran 2 - Sewa Gedung Pabrik` (201 Created)
- `16.8 POST Kalkulasi Alokasi Tarif Overhead Flat & Terbobot` (200 OK)
- `16.9 GET Laporan Ringkasan Alokasi Overhead Bulanan` (200 OK)
- `16.10 POST Kunci Periode Overhead Bulanan (Lock & Immutable)` (200 OK)
- `16.11 GET Laporan HPP / COGS Komprehensif Order (BPMN 01)` (200 OK)

---

## 6. Kesiapan Menuju Tahap 7.2

Dengan rampungnya Tahap 7.1, seluruh modul inti pada arsitektur Lukla Operations Engine telah selesai diimplementasikan. Tahap berikutnya adalah **Tahap 7.2: End-to-End Testing Suite & 4 SRS Business Scenarios Acceptance**, yang mencakup:
1. Validasi rantai proses terintegrasi dari Hulu ke Hilir (Master Data $\rightarrow$ Sample Prototype $\rightarrow$ Client Approval $\rightarrow$ Warehouse Cutting Allocation $\rightarrow$ Operator Sewing Output $\rightarrow$ Payroll Locking $\rightarrow$ QC Inspection & Bonus $\rightarrow$ Factory Overhead Allocation $\rightarrow$ Final Order Completion & COGS Analysis).
2. Verifikasi otomatis terhadap 4 skenario absolut SRS:
   - Skenario 1: Blokade Scrap Kain Melebihi Toleransi 5%.
   - Skenario 2: Payroll Operator & Lock Period dengan Piece-Rate Komponen.
   - Skenario 3: Pool QC Bonus 3.000 Pcs & Batas Cacat 2.5%.
   - Skenario 4: Sample v1 $\rightarrow$ v2 Approval $\rightarrow$ Konversi Order Massal 2.000 Pcs.

