# Laporan Hasil Pengerjaan: Tahap 2.2
## Penguncian Periode (Locking), Audit Adjustment, & Rekapitulasi Pelaporan (Governance & Summary)

- **Tanggal Selesai**: 14 Oktober 2026
- **Status**: Selesai 100% (Completed & Verified)
- **Modul**: `internal/modules/payroll` (BPMN 07)
- **Database Migration**: Version 25 (`000025_create_payroll_adjustments_table`)
- **Referensi Spesifikasi**:
  - `openapi-phase2-payroll.yaml`
  - `urutan pengerjaan.txt` (Tahap 2 - Payroll)
  - `Lukla_BPMN_v2.drawio` (Diagram 7: Payroll Mingguan - Borongan & Uang Makan)

---

## 1. Latar Belakang & Tujuan Bisnis

Tahap 2.2 melengkapi siklus hidup payroll mingguan penjahit borongan yang telah dihitung pada Tahap 2.1:
1. **Pencegahan Fraud & Rekalkulasi Liar (Governance / Locking)**: Periode payroll yang telah ditinjau bersama oleh Owner dan Finance harus dapat dikunci secara permanen (`LOCKED`). Setelah berstatus `LOCKED`, seluruh slip gaji menjadi *immutable* dan upaya kalkulasi ulang melalui engine otomatis harus ditolak.
2. **Audit Trail Penyesuaian (Adjustments)**: Segala bentuk penambahan bonus (misal: lembur darurat, insentif performa) maupun pemotongan (misal: kasbon, penalti disiplin) pasca-kalkulasi atau pasca-penguncian wajib melalui mekanisme formal audit adjustment yang mencatat jenis penyesuaian, nominal, alasan wajib (`reason`), dan identitas pengguna pembuat (`created_by_user_id`).
3. **Rekapitulasi Pelaporan Eksekutif (Summary Report)**: Menyediakan data ringkasan siap bayar bagi Finance dan Owner yang mengagregasikan total upah borongan, total uang makan, total penyesuaian, grand total kas/transfer bank, serta breakdown per line produksi dan per penjahit.

---

## 2. Perubahan Skema Database (Migration 25)

File migrasi yang diterapkan:
- **Up**: `migrations/db/migrations/000025_create_payroll_adjustments_table.up.sql`
- **Down**: `migrations/db/migrations/000025_create_payroll_adjustments_table.down.sql`

```sql
CREATE TABLE IF NOT EXISTS payroll_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    adjustment_type VARCHAR(20) NOT NULL CHECK (adjustment_type IN ('BONUS', 'DEDUCTION', 'CORRECTION', 'REVERSAL')),
    amount NUMERIC(15, 2) NOT NULL CHECK (amount <> 0),
    reason TEXT NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payroll_adjustments_period ON payroll_adjustments(payroll_period_id);
CREATE INDEX IF NOT EXISTS idx_payroll_adjustments_operator ON payroll_adjustments(operator_id);
```

---

## 3. Rincian Implementasi Kode

### A. Domain Model (`internal/modules/payroll/model.go`)
- Error constants baru:
  - `ErrPeriodMustBeCalculatedToLock`: Periode harus berstatus `CALCULATED` sebelum dikunci.
  - `ErrPeriodDraftCannotBeLocked`: Periode `DRAFT` tidak bisa dikunci.
  - `ErrInvalidAdjustmentType`: Jenis adjustment harus `BONUS`, `DEDUCTION`, `CORRECTION`, atau `REVERSAL`.
  - `ErrAdjustmentReasonRequired`: Alasan wajib diisi untuk kepatuhan audit.
  - `ErrAdjustmentZeroAmount`: Nominal tidak boleh 0.
- Struct model:
  - `PayrollAdjustment`
  - `CreateAdjustmentRequest`
  - `LockPayrollPeriodRequest`
  - `LinePayrollSummary`
  - `PayrollSummaryReport`

### B. Repository Layer (`internal/modules/payroll/postgres_repository.go`)
- **`LockPeriod(ctx, periodID, lockedByUserID, notes)`**:
  - Mengunci row periode dengan `SELECT ... FOR UPDATE`.
  - Memverifikasi status saat ini adalah `CALCULATED` (menolak `DRAFT` atau yang sudah `LOCKED`).
  - Memperbarui `status = 'LOCKED'`, `locked_at = CURRENT_TIMESTAMP`, `locked_by_user_id`, dan `notes`.
  - Memperbarui status seluruh slip pada periode terkait menjadi `LOCKED`.
- **`CreateAdjustment(ctx, periodID, adj)`**:
  - Memasukkan record audit ke `payroll_adjustments`.
  - Menyesuaikan `adjustments_total` dan `net_payable_total` pada `payroll_slips` milik operator.
  - Menyesuaikan `total_adjustments` dan `total_net_payable` pada `payroll_periods`.
- **`ListAdjustmentsByPeriod(ctx, periodID)`**:
  - Mengambil seluruh penyesuaian untuk periode beserta nama operator dan nama user pencatat.
- **`GetPayrollSummaryReport(ctx, periodID)`**:
  - Mengagregasi summary periode, grouping per line produksi, dan daftar operator summary.

### C. Service Layer (`internal/modules/payroll/service.go`)
- Memvalidasi RBAC (`owner_executive`, `finance_payroll`).
- Normalisasi tanda nominal (nilai deduction / reversal otomatis dijadikan nilai negatif).
- Validasi keberadaan alasan wajib (`reason`).

### D. Handler Layer (`internal/modules/payroll/handler.go`)
Mendaftarkan 4 endpoint baru:
1. `POST /api/v1/payroll/periods/:id/lock`
2. `GET /api/v1/payroll/periods/:id/adjustments`
3. `POST /api/v1/payroll/periods/:id/adjustments`
4. `GET /api/v1/payroll/periods/:id/summary`

---

## 4. Endpoints & Contoh Response

### 1. Lock Payroll Period (Finalize)
- **Method / URL**: `POST /api/v1/payroll/periods/:id/lock`
- **Request Body**:
```json
{
  "notes": "Periode payroll telah diaudit dan disetujui oleh Owner dan Finance. Data kini terkunci (immutable)."
}
```
- **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Payroll period PAY-2026-W42 locked successfully. Data is now finalized.",
  "data": {
    "id": "a8e9d3b2-6012-4c5b-91d8-ff0327ab4c10",
    "periodCode": "PAY-2026-W42",
    "startDate": "2026-10-12",
    "endDate": "2026-10-18",
    "mealAllowancePercentage": 12.5,
    "status": "LOCKED",
    "totalOperatorsCount": 24,
    "totalGrossPieceRate": 18500000,
    "totalMealAllowance": 2312500,
    "totalAdjustments": 50000,
    "totalNetPayable": 20862500,
    "lockedAt": "2026-10-14T18:00:00Z",
    "lockedByUserId": "3c1e2f4a-8b9d-4e5f-a012-3456789abcde",
    "notes": "Periode payroll telah diaudit dan disetujui oleh Owner dan Finance. Data kini terkunci (immutable)."
  }
}
```

### 2. Create Payroll Adjustment
- **Method / URL**: `POST /api/v1/payroll/periods/:id/adjustments`
- **Request Body**:
```json
{
  "operatorId": "6c7e145f-149b-430c-b26a-d46777bc3db5",
  "adjustmentType": "BONUS",
  "amount": 50000.0,
  "reason": "Bonus insentif kecepatan & ketepatan jahitan line 1"
}
```
- **Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "id": "9b1c2d3e-4f5a-6b7c-8d9e-0f1a2b3c4d5e",
    "payrollPeriodId": "a8e9d3b2-6012-4c5b-91d8-ff0327ab4c10",
    "operatorId": "6c7e145f-149b-430c-b26a-d46777bc3db5",
    "operatorName": "Siti Rahmawati",
    "adjustmentType": "BONUS",
    "amount": 50000,
    "reason": "Bonus insentif kecepatan & ketepatan jahitan line 1",
    "createdByUserId": "3c1e2f4a-8b9d-4e5f-a012-3456789abcde",
    "createdByName": "Finance Officer",
    "createdAt": "2026-10-14T17:45:00Z"
  }
}
```

### 3. List Payroll Adjustments
- **Method / URL**: `GET /api/v1/payroll/periods/:id/adjustments`
- **Response (200 OK)**:
```json
{
  "success": true,
  "data": [
    {
      "id": "9b1c2d3e-4f5a-6b7c-8d9e-0f1a2b3c4d5e",
      "payrollPeriodId": "a8e9d3b2-6012-4c5b-91d8-ff0327ab4c10",
      "operatorId": "6c7e145f-149b-430c-b26a-d46777bc3db5",
      "operatorName": "Siti Rahmawati",
      "adjustmentType": "BONUS",
      "amount": 50000,
      "reason": "Bonus insentif kecepatan & ketepatan jahitan line 1",
      "createdByUserId": "3c1e2f4a-8b9d-4e5f-a012-3456789abcde",
      "createdByName": "Finance Officer",
      "createdAt": "2026-10-14T17:45:00Z"
    }
  ]
}
```

### 4. Get Payroll Summary Report (Rekapitulasi Pelaporan)
- **Method / URL**: `GET /api/v1/payroll/periods/:id/summary`
- **Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "periodId": "a8e9d3b2-6012-4c5b-91d8-ff0327ab4c10",
    "periodCode": "PAY-2026-W42",
    "startDate": "2026-10-12",
    "endDate": "2026-10-18",
    "mealAllowancePercentage": 12.5,
    "status": "LOCKED",
    "totalOperators": 24,
    "totalPiecesProduced": 7400,
    "totalGrossPieceRate": 18500000,
    "totalMealAllowance": 2312500,
    "totalAdjustments": 50000,
    "grandTotalPayable": 20862500,
    "linesSummary": [
      {
        "lineId": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
        "lineCode": "LINE-01",
        "lineName": "Line Produksi Ciwidey 1",
        "operatorsCount": 24,
        "piecesProduced": 7400,
        "grossAmount": 18500000
      }
    ],
    "operatorsSummary": [
      {
        "id": "7e91f034-31da-42ea-a41d-cb092a6b2191",
        "payrollPeriodId": "a8e9d3b2-6012-4c5b-91d8-ff0327ab4c10",
        "operatorId": "6c7e145f-149b-430c-b26a-d46777bc3db5",
        "operatorCode": "OP-001",
        "operatorName": "Siti Rahmawati",
        "operatorJoinDate": "2026-08-01",
        "totalPieces": 380,
        "grossPieceRateTotal": 950000,
        "mealAllowanceAmount": 118750,
        "adjustmentsTotal": 50000,
        "netPayableTotal": 1118750,
        "status": "LOCKED",
        "createdAt": "2026-10-14T17:30:00Z",
        "updatedAt": "2026-10-14T18:00:00Z"
      }
    ]
  }
}
```

---

## 5. Hasil Verifikasi & Pengujian Otomatis

1. **Payroll Unit Tests** (`internal/modules/payroll/handler_test.go`):
   - `TestLockPeriod_Success`: PASS
   - `TestLockPeriod_DraftPeriod_Rejected`: PASS
   - `TestLockPeriod_AlreadyLocked_Conflict`: PASS
   - `TestLockPeriod_UnauthorizedRole`: PASS
   - `TestCreateAdjustment_Bonus_Success`: PASS
   - `TestCreateAdjustment_Deduction_NegativeAmount`: PASS
   - `TestCreateAdjustment_InvalidType_Rejected`: PASS
   - `TestCreateAdjustment_MissingReason_Rejected`: PASS
   - `TestCreateAdjustment_ZeroAmount_Rejected`: PASS
   - `TestListAdjustments_Success`: PASS
   - `TestGetPayrollSummary_Success`: PASS
   - **Total**: 24 tests PASS.

2. **Live DB Acceptance Test** (`internal/app/app_test.go`):
   - `TestApp_PayrollLockingAndAdjustmentLiveDB`: **PASS dalam 0.65 detik**.
   - Memverifikasi penguncian database secara riil, kalkulasi penyesuaian bonus & potongan, pengubahan status slip menjadi `LOCKED`, serta pemblokiran mutlak terhadap panggilan kalkulasi ulang (`409 Conflict`).

3. **Postman Collection**:
   - `Lukla_Articles_and_Components.postman_collection.json` folder `08. Tahap 2: Payroll Mingguan - Borongan & Uang Makan (BPMN 07)` kini memuat 10 request lengkap dengan test scripts dan contoh response.

