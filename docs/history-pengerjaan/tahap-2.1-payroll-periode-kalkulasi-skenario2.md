# Laporan Hasil Pengerjaan: Tahap 2.1
## Periode Payroll, Kalkulasi Borongan & Uang Makan (Core Engine & Skenario Uji 2)

- **Tanggal Selesai**: 14 Oktober 2026
- **Status**: Selesai 100% (Completed & Verified)
- **Modul**: `internal/modules/payroll` (BPMN 07)
- **Database Migration**: Version 24 (`000024_create_payroll_tables`)
- **Referensi Spesifikasi**:
  - `openapi-phase2-payroll.yaml`
  - `urutan pengerjaan.txt` (Tahap 2 - Payroll)
  - `Lukla_BPMN_v2.drawio` (Diagram 7: Payroll Mingguan - Borongan & Uang Makan)
  - **Skenario Uji 2 SRS**: Penjahit baru bergabung di tengah minggu.

---

## 1. Latar Belakang & Tujuan Bisnis

Tahap 2.1 berfokus pada pembangunan inti mesin kalkulasi upah (*payroll engine*) mingguan untuk seluruh penjahit borongan di pabrik konveksi Lukla:
1. **Pengelolaan Periode Payroll**: Pembuatan siklus mingguan (`start_date` s.d. `end_date`), penentuan persentase uang makan (10.00% – 15.00%), dan status periode (`DRAFT`, `CALCULATED`, `LOCKED`).
2. **Kalkulasi Borongan Presisi**: Mengagregasikan `production_outputs` terverifikasi mandor dan mengalikan kuantitas output dengan `snapshot_sewing_rate` komponen yang berlaku saat output dicatat.
3. **Uang Makan Proporsional**: Mengalokasikan uang makan sebesar persentase yang ditetapkan dari total pendapatan borongan penjahit.
4. **Validasi Skenario Uji 2**: Penjahit yang baru bergabung di tengah minggu (misal hari Rabu) **hanya boleh dihitung outputnya sejak tanggal mulai bekerja (`join_date`)**. Segala output sebelum tanggal bergabung wajib diabaikan secara ketat oleh engine untuk mencegah salah bayar.
5. **Idempotensi Kalkulasi**: Endpoint kalkulasi (`POST /calculate`) dapat dipanggil berulang kali secara aman selama periode berstatus `DRAFT` atau `CALCULATED`, melakukan pembersihan slip lama dan menghitung ulang snapshot data output produksi terbaru secara deterministik.

---

## 2. Perubahan Skema Database (Migration 24)

File migrasi yang diterapkan:
- **Up**: `migrations/db/migrations/000024_create_payroll_tables.up.sql`
- **Down**: `migrations/db/migrations/000024_create_payroll_tables.down.sql`

```sql
CREATE TABLE IF NOT EXISTS payroll_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_code VARCHAR(50) UNIQUE NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    meal_allowance_percentage NUMERIC(5, 2) NOT NULL CHECK (meal_allowance_percentage >= 10.00 AND meal_allowance_percentage <= 15.00),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'CALCULATED', 'LOCKED')),
    total_operators_count INT NOT NULL DEFAULT 0,
    total_gross_piece_rate NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_meal_allowance NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_adjustments NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_net_payable NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    locked_at TIMESTAMPTZ,
    locked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_period_dates CHECK (start_date <= end_date)
);

CREATE TABLE IF NOT EXISTS payroll_slips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    total_pieces INT NOT NULL DEFAULT 0,
    gross_piece_rate_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    meal_allowance_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    adjustments_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    net_payable_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'CALCULATED' CHECK (status IN ('CALCULATED', 'LOCKED', 'PAID')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_payroll_period_operator UNIQUE (payroll_period_id, operator_id)
);

CREATE TABLE IF NOT EXISTS payroll_slip_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_slip_id UUID NOT NULL REFERENCES payroll_slips(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    work_date DATE NOT NULL,
    quantity INT NOT NULL,
    snapshot_sewing_rate NUMERIC(12, 2) NOT NULL,
    subtotal NUMERIC(15, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 3. Rincian Implementasi Kode

1. **Struktur Paket Baru**: `internal/modules/payroll`
   - `model.go`: Mendefinisikan struct `PayrollPeriod`, `PayrollSlip`, `PayrollSlipDetail`, `PayrollSlipItem`, `OperatorOutputAggregation`.
   - `repository.go`: Kontrak antarmuka `Repository`.
   - `postgres_repository.go`:
     - `GetEligibleOperatorOutputs`: Query cerdas yang menggabungkan `production_outputs`, `operators`, `orders`, `clients`, dan `components`, serta menerapkan filter kualifikasi `po.work_date >= op.join_date`.
     - `SaveCalculationResults`: Eksekusi atomik dalam transaksi `BeginTx` yang menghapus slip lama dan menyimpan data slip beserta seluruh rincian item baru secara terstruktur.
   - `service.go`: Menghitung agregasi nominal, pembulatan matematika presisi dua desimal (`math.Round`), dan verifikasi hak akses peran (`owner_executive`, `finance_payroll`).
   - `handler.go`: Endpoint HTTP RESTful Gin.

2. **Daftar Endpoint Tahap 2.1**:
   - `GET /api/v1/payroll/periods`: Menampilkan daftar periode payroll dengan filter status dan pagination.
   - `POST /api/v1/payroll/periods`: Membuat periode payroll baru (status awal `DRAFT`).
   - `GET /api/v1/payroll/periods/:id`: Mengambil detail satu periode payroll.
   - `POST /api/v1/payroll/periods/:id/calculate`: Menjalankan kalkulasi upah borongan dan uang makan.
   - `GET /api/v1/payroll/periods/:id/slips`: Mengambil daftar slip gaji penjahit pada periode tertentu.
   - `GET /api/v1/payroll/slips/:id`: Mengambil rincian breakdown slip per item pekerjaan, tarif jahit snapshot, dan uang makan.

---

## 4. Hasil Verifikasi & Pengujian Otomatis

1. **Unit Testing** (`internal/modules/payroll/handler_test.go`):
   - 14 test cases awal lulus 100% (validasi rentang persentase 10-15%, proteksi otorisasi peran, validasi rentang tanggal).
2. **Live DB Acceptance Test Skenario Uji 2** (`TestApp_PayrollSkenario2LiveDB` di `app_test.go`):
   - Dibuat operator baru `Dewi Penjahit Baru Midweek` dengan `join_date = 2026-10-14` (Rabu).
   - Diinputkan output produksi:
     - Senin (2026-10-12): 50 pcs @ Rp 3.000 (sebelum join date)
     - Rabu (2026-10-14): 80 pcs @ Rp 3.000
     - Kamis (2026-10-15): 120 pcs @ Rp 3.000
   - Hasil kalkulasi engine:
     - Output hari Senin **secara tepat diabaikan (EXCLUDED)**.
     - Total pieces terhitung: **200 pcs** (80 + 120).
     - Upah borongan kotor: **Rp 600.000,00** (200 × 3.000).
     - Uang makan 12.5%: **Rp 75.000,00**.
     - Total bersih: **Rp 675.000,00**.
   - Pengujian lulus dalam **0.91 detik**.

