-- Migration 000033: Create Factory Overhead Tables (BPMN 09)

CREATE TABLE IF NOT EXISTS overhead_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS overhead_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_code VARCHAR(30) NOT NULL UNIQUE,
    period_month INT NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year INT NOT NULL CHECK (period_year BETWEEN 2024 AND 2050),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CALCULATED', 'LOCKED')),
    total_expense_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_produced_units INT NOT NULL DEFAULT 0,
    garment_units INT NOT NULL DEFAULT 0,
    bag_units INT NOT NULL DEFAULT 0,
    weight_garment NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    weight_bag NUMERIC(5, 2) NOT NULL DEFAULT 1.25,
    flat_rate_per_unit NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    garment_rate_per_unit NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    bag_rate_per_unit NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    notes TEXT,
    calculated_at TIMESTAMPTZ,
    locked_at TIMESTAMPTZ,
    locked_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_overhead_period_month_year UNIQUE (period_year, period_month)
);

CREATE TABLE IF NOT EXISTS overhead_expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    overhead_period_id UUID NOT NULL REFERENCES overhead_periods(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES overhead_categories(id),
    expense_date DATE NOT NULL,
    amount NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    invoice_reference VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS overhead_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    overhead_period_id UUID NOT NULL REFERENCES overhead_periods(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    article_category VARCHAR(50) NOT NULL,
    produced_units INT NOT NULL DEFAULT 0,
    applied_rate_per_unit NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    allocated_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_overhead_allocation_period_order UNIQUE (overhead_period_id, order_id)
);

CREATE INDEX IF NOT EXISTS idx_overhead_periods_status ON overhead_periods(status);
CREATE INDEX IF NOT EXISTS idx_overhead_expenses_period ON overhead_expenses(overhead_period_id);
CREATE INDEX IF NOT EXISTS idx_overhead_allocations_order ON overhead_allocations(order_id);

-- Seed default overhead categories
INSERT INTO overhead_categories (code, name, description)
VALUES 
    ('ELECTRICITY', 'Listrik PLN Pabrik', 'Biaya konsumsi daya listrik mesin jahit, kompresor dan penerangan lini produksi'),
    ('WATER', 'Air & Sanitasi Pabrik', 'Biaya utilitas air bersih dan sanitasi workshop'),
    ('INTERNET', 'Internet & Telekomunikasi Operasional', 'Biaya jaringan internet WiFi dan komunikasi operasional'),
    ('FACTORY_RENT', 'Sewa Gedung Pabrik & Workshop', 'Alokasi bulanan sewa tempat workshop manufaktur'),
    ('MACHINE_MAINTENANCE', 'Pemeliharaan Mesin & Spareparts', 'Servis rutin mesin jahit, penggantian jarum, sekoci, dan minyak mesin'),
    ('INDIRECT_MATERIALS', 'Bahan Pembantu & ATK Produksi', 'Kertas pola, kapur jahit, plastik pelindung, dan ATK mandor/QC'),
    ('OPERATIONAL_OTHERS', 'Operasional Umum Lainnya', 'Biaya pendukung kebersihan, keamanan, dan logistik lokal')
ON CONFLICT (code) DO NOTHING;

