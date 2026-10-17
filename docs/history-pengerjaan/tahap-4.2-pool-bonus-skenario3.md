# Laporan Pengerjaan Tahap 4.2: QC Pool Bonus 3-Arah & Skenario Uji 3 (BPMN 08 / FR-10.3)

**Tanggal Penyelesaian:** 15 Oktober 2026  
**Status:** Selesai (100% Terverifikasi Otomatis pada Live Database)  
**Database Migration:** `000030_create_qc_bonus_tables.up.sql` (DB Version: 30)  
**Target Modul:** `internal/modules/qcbonus`, `internal/app`  

---

## 1. Latar Belakang & Kebutuhan Bisnis

Tahap 4.2 merupakan implementasi dari alur pembagian bonus pengawasan mutu (*Quality Control Pool Bonus*) sesuai dengan diagram proses bisnis **BPMN 08** dan ketentuan spesifikasi kebutuhan sistem SRS **FR-10.3 (Skenario Uji 3)**.

### Aturan Bisnis & Kontrol Akses:
1. **Hak Eksklusif Pemilik Bisnis (`OWNER` / `owner_executive`):**
   - Tarif bonus per pcs produk yang lolos QC dikontrol secara eksklusif oleh Pemilik Bisnis (Owner).
   - Default tarif standar pabrik adalah **Rp 1.500 / pcs**.
   - Setiap upaya pengubahan tarif oleh pengguna non-owner (misal Mandor, Penjahit, Gudang, atau QC Inspector) ditolak tegas dengan kode status **`403 Forbidden`**.
   - Setiap pembaruan tarif otomatis dicatat pada tabel riwayat (`qc_bonus_rate_history`) lengkap dengan timestamp, user pengubah, dan catatan alasan perubahan.
2. **Penerima Hak Pool Bonus 3-Arah:**
   - Sesuai rancangan proses BPMN 08, pool bonus dialokasikan rata kepada pengawas lantai produksi: **2 Mandor Line + 1 Kepala Operasional / PIC Line** (total 3 orang).
   - Rasio pembagian default adalah sepertiga rata per penerima ($1/3 \approx 0.3333333333333333$).
3. **Siklus Periode & Integritas Data (*State Lifecycle*):**
   - `OPEN` / `DRAFT` $\to$ Periode bonus dibuka untuk rentang tanggal tertentu (misal mingguan).
   - `CALCULATED` $\to$ Sistem mengagregasi seluruh produk yang dinyatakan lolos inspeksi QC (`passed_quantity`) pada rentang tanggal tersebut, mengalikannya dengan tarif aktif, lalu mendistribusikan nominalnya ke masing-masing penerima.
   - `LOCKED` $\to$ Periode dikunci setelah disetujui Owner. Seluruh data terkunci menjadi *immutable*. Jika sistem menerima instruksi perhitungan ulang terhadap periode yang berstatus `LOCKED`, request ditolak dengan kode status **`409 Conflict`**.
4. **Verifikasi Matematis Skenario Uji 3 (SRS FR-10.3):**
   - Akumulasi kuantitas produk lolos QC: **3.000 pcs**.
   - Tarif per pcs: **Rp 1.500**.
   - Total dana pool bonus QC: $3.000 \times \text{Rp } 1.500 = \textbf{Rp 4.500.000}$.
   - Pembagian hak bonus tepat sepertiga rata kepada 3 orang tanpa sisa sen yang hilang:
     $$\frac{\text{Rp } 4.500.000}{3} = \textbf{Rp 1.500.000 / orang}$$

---

## 2. Perubahan Skema Database

File migrasi yang diterapkan ke live database PostgreSQL (`lukla_db`):
- `migrations/db/migrations/000030_create_qc_bonus_tables.up.sql`
- `migrations/db/migrations/000030_create_qc_bonus_tables.down.sql`

### Rincian Tabel Baru:
1. **`qc_bonus_configs` (Konfigurasi Tarif Bonus QC):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `rate_per_pcs` (NUMERIC(12,2), NOT NULL, DEFAULT 1500.00, CHECK `rate_per_pcs >= 0`)
   - `currency` (VARCHAR(10), NOT NULL, DEFAULT 'IDR')
   - `updated_by_user_id` (UUID, NOT NULL, FK `users(id)`)
   - `notes` (TEXT)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
2. **`qc_bonus_rate_history` (Riwayat Perubahan Tarif):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `rate_per_pcs` (NUMERIC(12,2), NOT NULL)
   - `changed_by_user_id` (UUID, NOT NULL, FK `users(id)`)
   - `effective_from` (TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)
   - `notes` (TEXT)
   - `created_at` (TIMESTAMPTZ)
3. **`qc_bonus_recipients` (Daftar Pengawas Penerima Bonus Pool):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `user_id` (UUID, NOT NULL, UNIQUE, FK `users(id)`)
   - `is_active` (BOOLEAN, DEFAULT TRUE)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
   - *Seeded Pengawas Default (3 Orang):*
     - Mandor 1: `Bambang Supriyadi` (`e6447b6c-7ffc-462f-a0d6-a2ef6a387634`)
     - Mandor 2: `Cecep Hidayat` (`3af11b6e-ed26-4ff7-b91a-6246b3924ad0`)
     - Kepala Operasional / Produksi: `Dadang Kurnia` (`f5d8de29-e94c-441a-a124-96b2540e2c56`)
4. **`qc_bonus_periods` (Siklus Periode Pool Bonus QC):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `name` (VARCHAR(100), NOT NULL)
   - `start_date` (DATE, NOT NULL)
   - `end_date` (DATE, NOT NULL, CHECK `end_date >= start_date`)
   - `status` (VARCHAR(20), NOT NULL, DEFAULT 'DRAFT', CHECK `status IN ('DRAFT', 'OPEN', 'CALCULATED', 'LOCKED')`)
   - `rate_per_pcs_applied` (NUMERIC(12,2), NOT NULL, DEFAULT 0)
   - `total_passed_pcs` (INT, NOT NULL, DEFAULT 0)
   - `total_pool_amount` (NUMERIC(14,2), NOT NULL, DEFAULT 0)
   - `recipient_count` (INT, NOT NULL, DEFAULT 0)
   - `amount_per_recipient` (NUMERIC(14,2), NOT NULL, DEFAULT 0)
   - `notes` (TEXT)
   - `locked_at` (TIMESTAMPTZ)
   - `locked_by_user_id` (UUID, FK `users(id)`)
   - `created_at`, `updated_at` (TIMESTAMPTZ)
5. **`qc_bonus_distributions` (Distribusi Hak Pembagian per Pengawas):**
   - `id` (UUID PK, default `gen_random_uuid()`)
   - `period_id` (UUID, NOT NULL, FK `qc_bonus_periods(id)` ON DELETE CASCADE)
   - `user_id` (UUID, NOT NULL, FK `users(id)`)
   - `share_ratio` (NUMERIC(8,6), NOT NULL)
   - `share_amount` (NUMERIC(14,2), NOT NULL, CHECK `share_amount >= 0`)
   - `payout_status` (VARCHAR(30), NOT NULL, DEFAULT 'PENDING_DISBURSEMENT')
   - `created_at` (TIMESTAMPTZ)
   - *Constraint:* `UNIQUE(period_id, user_id)`

---

## 3. Implementasi Modul Go (`internal/modules/qcbonus`)

### A. Komponen Kode:
1. **Model (`internal/modules/qcbonus/model.go`):**
   - Struct data: `QCBonusConfig`, `QCBonusRateHistoryItem`, `UpdateQCBonusRateRequest`, `QCBonusRecipient`, `UpdateQCBonusRecipientsRequest`, `QCBonusPeriodSummary`, `CreateQCBonusPeriodRequest`, `QCBonusPeriodDetail`, `QCBonusDistributionItem`, `CalculateQCBonusResponse`, `LockQCBonusResponse`.
   - Error sentinel: `ErrBonusConfigNotFound`, `ErrBonusPeriodNotFound`, `ErrPeriodAlreadyLocked`, `ErrInvalidBonusRate`, `ErrInvalidPeriodDates`, `ErrNoActiveRecipients`, `ErrForbiddenQualityBonus`, `ErrForbiddenOwnerOnly`.
2. **Repository (`internal/modules/qcbonus/repository.go` & `postgres_repository.go`):**
   - `GetBonusConfig` & `UpdateBonusRate`: Mengelola konfigurasi tarif aktif dengan pencatatan audit trail otomatis.
   - `ListRecipients` & `UpdateRecipients`: Mengelola daftar penerima aktif serta menghitung `shareRatio` secara dinamis ($1 / N$).
   - `CreatePeriod`, `ListPeriods`, `GetPeriodByID`.
   - `CalculatePeriod`: Agregasi kuantitas lolos QC (`passed_quantity`) dari tabel `qc_inspections` dalam rentang `start_date` s.d. `end_date`, pengalian tarif, serta implementasi *Penny Allocation Algorithm* untuk membagi rata total pool ke seluruh pengawas tanpa sisa pembulatan sen/rupiah.
   - `LockPeriod`: Mengunci periode ke status `LOCKED`, mencatat `locked_at` dan `locked_by_user_id`.
3. **Service Layer (`internal/modules/qcbonus/service.go`):**
   - Penegakan RBAC ketat: Penolakan akses konfigurasi tarif jika pengguna bukan berhak peran `owner_executive`.
   - Pengawalan siklus status periode (*guard against recalculating locked period*).
4. **Handler Layer (`internal/modules/qcbonus/handler.go`):**
   - Registrasi endpoint router Gin:
     - `GET /api/v1/qc-bonus/config`
     - `PUT /api/v1/qc-bonus/config`
     - `GET /api/v1/qc-bonus/recipients`
     - `PUT /api/v1/qc-bonus/recipients`
     - `GET /api/v1/qc-bonus/periods`
     - `POST /api/v1/qc-bonus/periods`
     - `GET /api/v1/qc-bonus/periods/:id`
     - `POST /api/v1/qc-bonus/periods/:id/calculate`
     - `POST /api/v1/qc-bonus/periods/:id/lock`

---

## 4. Hasil Pengujian Otomatis & Skenario Uji 3

### A. Unit Testing (`internal/modules/qcbonus/qcbonus_test.go`)
- `TestQCBonusService_OwnerRateGuard`: Memverifikasi bahwa non-owner ditolak `403 Forbidden` saat mengubah tarif, sedangkan owner berhasil.
- `TestQCBonusService_CalculatePeriod_LockGuard`: Memverifikasi bahwa periode yang sudah `LOCKED` menolak perhitungan ulang dengan `409 Conflict`.
- `TestQCBonus_PennyAllocationAlgorithm`: Memverifikasi pembagian matematis pecahan ganjil (misal Rp 100.000 dibagi 3 orang $\to$ 33.334 + 33.333 + 33.333 = 100.000 pas).

### B. Acceptance Test Live Database (`TestApp_QCBonusPoolLiveDB` di `internal/app/app_test.go`)
Hasil pengujian end-to-end pada database PostgreSQL riil:
1. **Konfigurasi Tarif:**
   - Non-owner (`production@lukla.local`) mencoba mengubah tarif $\to$ **403 Forbidden**.
   - Owner (`executive@lukla.local`) menetapkan tarif Rp 1.500 / pcs $\to$ **200 OK**.
2. **Daftar Penerima:**
   - Mengambil 3 penerima bawaan (2 Mandor + 1 Kepala Produksi) dengan `shareRatio` masing-masing 0.333333.
3. **Pembukaan Periode:**
   - Buka periode `2026-11-01` s.d. `2026-11-07` dengan status awal `DRAFT`.
4. **Eksekusi Skenario Uji 3 (SRS FR-10.3):**
   - Seeding data riil inspeksi QC lolos masal: **3.000 pcs**.
   - Trigger endpoint kalkulasi: `POST /api/v1/qc-bonus/periods/:id/calculate`.
   - **Hasil Aktual:**
     - `totalPassedPcs`: `3000`
     - `ratePerPcsApplied`: `1500.00`
     - `totalPoolAmount`: `4500000.00`
     - `recipientCount`: `3`
     - `amountPerRecipient`: `1500000.00`
     - Distribusi rincian:
       - Mandor 1 (`Bambang Supriyadi`): **Rp 1.500.000,00**
       - Mandor 2 (`Cecep Hidayat`): **Rp 1.500.000,00**
       - Kepala Operasional (`Dadang Kurnia`): **Rp 1.500.000,00**
       - Selisih total pembagian: **Rp 0,00 (Terbagi Sempurna Tanpa Sisa)**.
5. **Penguncian & Proteksi Mutasi:**
   - Trigger endpoint lock: `POST /api/v1/qc-bonus/periods/:id/lock` $\to$ status berubah menjadi `LOCKED`.
   - Panggilan ulang calculate pada periode yang sama $\to$ ditolak **409 Conflict**.

```text
=== RUN   TestApp_QCBonusPoolLiveDB
    app_test.go:3983: Distribusi Hak Penerima: Bambang Supriyadi (Kepala Produksi / Mandor) = Rp 1500000.00
    app_test.go:3983: Distribusi Hak Penerima: Cecep Hidayat (Kepala Produksi / Mandor) = Rp 1500000.00
    app_test.go:3983: Distribusi Hak Penerima: Dadang Kurnia (Kepala Produksi / Mandor) = Rp 1500000.00
    app_test.go:3988: VERIFIKASI SKENARIO UJI 3 SUKSES: 3.000 pcs lolos QC x Rp 1.500 = Rp 4.500.000 terbagi rata sempurna ke 3 orang (Rp 1.500.000/orang)
--- PASS: TestApp_QCBonusPoolLiveDB (0.78s)
PASS
```

Seluruh test suite aplikasi (`go test ./...`) menghasilkan status **100% PASS**:
```text
ok  	github.com/lukla/operations-engine/internal/app	15.454s
ok  	github.com/lukla/operations-engine/internal/modules/identity	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/inventory	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/masterdata	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/payroll	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/production	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/qcbonus	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/quality	(cached)
ok  	github.com/lukla/operations-engine/internal/modules/ratecard	(cached)
```

---

## 5. Sinkronisasi Postman Collection

Postman Collection [`Lukla_Articles_and_Components.postman_collection.json`](../Lukla_Articles_and_Components.postman_collection.json) telah disinkronkan dengan penambahan folder baru:
`11. Tahap 4.2: QC Pool Bonus 3-Arah & Skenario Uji 3 (BPMN 08)`

### Rincian Request Items & Contoh Response Riil:
1. `GET /qc-bonus/config`: Mengambil tarif aktif saat ini (Rp 1.500) dan riwayat perubahan (`{{base_url}}/qc-bonus/config`).
2. `PUT /qc-bonus/config` (Non-Owner): Simulasi penolakan `403 Forbidden` saat role bukan owner.
3. `PUT /qc-bonus/config` (Owner): Pemilik menetapkan tarif Rp 1.500 / pcs (`200 OK`).
4. `GET /qc-bonus/recipients`: Daftar 3 penerima bonus pool bawaan (`200 OK`).
5. `PUT /qc-bonus/recipients` (Non-Owner): Simulasi penolakan `403 Forbidden` saat role non-owner mengubah penerima.
6. `PUT /qc-bonus/recipients` (Owner): Pemilik memperbarui daftar 3 penerima bonus pool (`200 OK`).
7. `POST /qc-bonus/periods`: Membuka siklus periode baru berstatus `DRAFT` (`201 Created`).
8. `POST /qc-bonus/periods/:id/calculate`: Kalkulasi agregasi kuantitas lolos QC (3.000 pcs $\times$ Rp 1.500 = Rp 4.500.000) terbagi rata tepat Rp 1.500.000 / orang (`200 OK`).
9. `POST /qc-bonus/periods/:id/lock`: Mengunci periode bonus QC menjadi status `LOCKED` (`200 OK`).
10. `POST /qc-bonus/periods/:id/calculate`: Menolak kalkulasi ulang pada periode `LOCKED` (`409 Conflict`).
11. `GET /qc-bonus/periods/:id`: Mengambil detail lengkap periode dan distribusi hak (`200 OK`).
12. `GET /qc-bonus/periods`: Mengambil riwayat ringkasan seluruh periode bonus (`200 OK`).

