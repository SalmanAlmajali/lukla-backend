-- Migration 000028: Create Inventory Outbound Tables (Issues, Blocked Attempts, Cutting Scraps)
-- Skenario BPMN 02: Pengeluaran ke Meja Potong & Blokade Kontaminasi Silang (FR-6.2, FR-6.3, FR-6.4)

-- 1. Tabel Pengeluaran Bahan ke Lantai Cutting (Material Issues)
CREATE TABLE IF NOT EXISTS inventory_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_number VARCHAR(50) NOT NULL UNIQUE,
    target_order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    cutting_table_number VARCHAR(50),
    issue_date DATE NOT NULL,
    issued_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_issues_order ON inventory_issues(target_order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_issues_number ON inventory_issues(issue_number);
CREATE INDEX IF NOT EXISTS idx_inventory_issues_date ON inventory_issues(issue_date);

-- 2. Detail Item Pengeluaran Bahan (Issue Items)
CREATE TABLE IF NOT EXISTS inventory_issue_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES inventory_issues(id) ON DELETE CASCADE,
    lot_id UUID NOT NULL REFERENCES inventory_lots(id) ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    quantity NUMERIC(15, 2) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_issue_items_issue ON inventory_issue_items(issue_id);
CREATE INDEX IF NOT EXISTS idx_inventory_issue_items_lot ON inventory_issue_items(lot_id);

-- 3. Log Audit Percobaan Salah-Keluar yang Diblokir (Blocked Attempts Audit Log / Skenario Uji 1)
CREATE TABLE IF NOT EXISTS inventory_blocked_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    attempted_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    lot_id UUID NOT NULL REFERENCES inventory_lots(id) ON DELETE RESTRICT,
    lot_number VARCHAR(50) NOT NULL,
    lot_client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    target_order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    target_order_number VARCHAR(50) NOT NULL,
    target_client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    requested_quantity NUMERIC(15, 2) NOT NULL,
    reason TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_inventory_blocked_attempts_lot ON inventory_blocked_attempts(lot_id);
CREATE INDEX IF NOT EXISTS idx_inventory_blocked_attempts_order ON inventory_blocked_attempts(target_order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_blocked_attempts_date ON inventory_blocked_attempts(attempted_at);

-- 4. Tabel Pencatatan Sisa Potongan Kain / Limbah Cutting (Cutting Scraps - FR-6.3)
CREATE TABLE IF NOT EXISTS cutting_scraps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    scrap_category VARCHAR(50) NOT NULL CHECK (scrap_category IN ('PERCA_KECIL', 'CACAT_KAIN_ROLL', 'UJUNG_GULUNGAN', 'KESALAHAN_POTONG')),
    quantity NUMERIC(15, 2) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL,
    disposition VARCHAR(50) NOT NULL CHECK (disposition IN ('SIMPAN_GUDANG_SISA', 'KEMBALIKAN_KLIEN', 'JUAL_LIMBAH', 'BUANG')),
    reported_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cutting_scraps_order ON cutting_scraps(order_id);
CREATE INDEX IF NOT EXISTS idx_cutting_scraps_material ON cutting_scraps(material_id);
CREATE INDEX IF NOT EXISTS idx_cutting_scraps_category ON cutting_scraps(scrap_category);

