-- Migration 000030: Create QC Bonus Tables & Seed Default Configuration (BPMN 08 & Skenario Uji 3)

-- 1. Tabel Konfigurasi Tarif Bonus QC (Khusus Wewenang Owner)
CREATE TABLE IF NOT EXISTS qc_bonus_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_per_pcs NUMERIC(15, 2) NOT NULL DEFAULT 1500.00 CHECK (rate_per_pcs >= 0),
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    updated_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Histori Perubahan Tarif Bonus QC
CREATE TABLE IF NOT EXISTS qc_bonus_rate_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_per_pcs NUMERIC(15, 2) NOT NULL CHECK (rate_per_pcs >= 0),
    changed_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Daftar Personel Penerima Hak Pool Bonus QC (Default 3 Pengawas: 2 Mandor + 1 Kepala Produksi)
CREATE TABLE IF NOT EXISTS qc_bonus_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_qc_bonus_recipient_user UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_qc_bonus_recipients_user ON qc_bonus_recipients(user_id);
CREATE INDEX IF NOT EXISTS idx_qc_bonus_recipients_active ON qc_bonus_recipients(is_active);

-- 4. Tabel Periode Akumulasi Pool Bonus QC
CREATE TABLE IF NOT EXISTS qc_bonus_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'CALCULATED', 'LOCKED')),
    rate_per_pcs_applied NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_passed_pcs INT NOT NULL DEFAULT 0,
    total_pool_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    recipient_count INT NOT NULL DEFAULT 0,
    amount_per_recipient NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    locked_at TIMESTAMPTZ,
    locked_by_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_qc_bonus_period_dates CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_qc_bonus_periods_dates ON qc_bonus_periods(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_qc_bonus_periods_status ON qc_bonus_periods(status);

-- 5. Tabel Distribusi Pembagian Hak Bonus per Personel Pengawas (Bagi Rata ÷ 3 / Skenario Uji 3)
CREATE TABLE IF NOT EXISTS qc_bonus_distributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id UUID NOT NULL REFERENCES qc_bonus_periods(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    share_amount NUMERIC(15, 2) NOT NULL CHECK (share_amount >= 0),
    payout_status VARCHAR(30) NOT NULL DEFAULT 'PENDING_DISBURSEMENT' CHECK (payout_status IN ('PENDING_DISBURSEMENT', 'PAID', 'CANCELLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_qc_bonus_dist_period_user UNIQUE (period_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_qc_bonus_dist_period ON qc_bonus_distributions(period_id);
CREATE INDEX IF NOT EXISTS idx_qc_bonus_dist_user ON qc_bonus_distributions(user_id);

-- Seed Mandor Tambahan jika belum ada agar total pengawas menjadi 3 orang
INSERT INTO users (
    employee_code, full_name, email, phone_number, password_hash, role_id, can_access_app, is_active
) VALUES
    (
        'EMP-MDR-001',
        'Cecep Hidayat',
        'mandor1@lukla.local',
        '081122334408',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'production_head'),
        TRUE,
        TRUE
    ),
    (
        'EMP-MDR-002',
        'Dadang Kurnia',
        'mandor2@lukla.local',
        '081122334409',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        (SELECT id FROM roles WHERE code = 'production_head'),
        TRUE,
        TRUE
    )
ON CONFLICT (email) DO NOTHING;

-- Seed Konfigurasi Tarif Awal oleh Owner (Rp 1.500 / pcs)
INSERT INTO qc_bonus_configs (rate_per_pcs, currency, updated_by_user_id, notes)
VALUES (
    1500.00,
    'IDR',
    (SELECT id FROM users WHERE email = 'executive@lukla.local'),
    'Tarif default pool bonus QC per pcs lolos ditetapkan oleh Owner'
);

INSERT INTO qc_bonus_rate_history (rate_per_pcs, changed_by_user_id, notes)
VALUES (
    1500.00,
    (SELECT id FROM users WHERE email = 'executive@lukla.local'),
    'Inisialisasi tarif bonus QC awal'
);

-- Seed 3 Penerima Default (1 Kepala Produksi + 2 Mandor)
INSERT INTO qc_bonus_recipients (user_id, is_active)
SELECT id, TRUE FROM users WHERE email IN ('production@lukla.local', 'mandor1@lukla.local', 'mandor2@lukla.local')
ON CONFLICT (user_id) DO NOTHING;

