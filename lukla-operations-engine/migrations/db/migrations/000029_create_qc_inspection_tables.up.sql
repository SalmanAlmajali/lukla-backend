-- Migration 000029: Create QC Inspection Tables & Seed Defect Categories (BPMN 06)
-- Skenario BPMN 06: Pemeriksaan QC, Defect Tracking, dan Lolos/Repair/Reject

-- 1. Master Kategori Cacat Produk (QC Defect Categories)
CREATE TABLE IF NOT EXISTS qc_defect_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('MINOR', 'MAJOR', 'CRITICAL')),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_qc_defect_categories_code ON qc_defect_categories(code);
CREATE INDEX IF NOT EXISTS idx_qc_defect_categories_active ON qc_defect_categories(is_active);

-- Seed Default Defect Categories
INSERT INTO qc_defect_categories (code, name, severity, description, is_active)
VALUES
    ('DEF-SEW-01', 'Jahitan Loncat / Lepas', 'MAJOR', 'Jahitan rantai atau obras melompat lebih dari 3 tusukan', TRUE),
    ('DEF-WRI-01', 'Kerutan / Puckering', 'MINOR', 'Kerutan kain berlebih di sekitar sambungan jahitan', TRUE),
    ('DEF-STN-01', 'Noda Kain / Minyak Mesin', 'MAJOR', 'Terciprat pelumas mesin atau noda kotoran pada panel', TRUE),
    ('DEF-SIZ-01', 'Ukuran Meleset / Measurement Off', 'CRITICAL', 'Toleransi ukuran panjang/lingkar melebihi batas spek (+/- 1 cm)', TRUE),
    ('DEF-ACC-01', 'Aksesoris Rusak / Cacat', 'MAJOR', 'Resleting macet, kancing patah, atau tali serut sobek', TRUE)
ON CONFLICT (code) DO NOTHING;

-- 2. Transaksi Pemeriksaan QC (QC Inspections)
CREATE TABLE IF NOT EXISTS qc_inspections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspection_number VARCHAR(50) NOT NULL UNIQUE,
    assembly_id UUID NOT NULL REFERENCES assemblies(id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    inspector_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    inspection_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_checked INT NOT NULL CHECK (total_checked > 0),
    passed_quantity INT NOT NULL DEFAULT 0 CHECK (passed_quantity >= 0),
    repair_quantity INT NOT NULL DEFAULT 0 CHECK (repair_quantity >= 0),
    reject_quantity INT NOT NULL DEFAULT 0 CHECK (reject_quantity >= 0),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_qc_inspection_sum CHECK (total_checked = passed_quantity + repair_quantity + reject_quantity)
);

CREATE INDEX IF NOT EXISTS idx_qc_inspections_order ON qc_inspections(order_id);
CREATE INDEX IF NOT EXISTS idx_qc_inspections_assembly ON qc_inspections(assembly_id);
CREATE INDEX IF NOT EXISTS idx_qc_inspections_inspector ON qc_inspections(inspector_id);
CREATE INDEX IF NOT EXISTS idx_qc_inspections_date ON qc_inspections(inspection_date);
CREATE INDEX IF NOT EXISTS idx_qc_inspections_number ON qc_inspections(inspection_number);

-- 3. Detail Temuan Cacat Pemeriksaan QC (QC Inspection Defects)
CREATE TABLE IF NOT EXISTS qc_inspection_defects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspection_id UUID NOT NULL REFERENCES qc_inspections(id) ON DELETE CASCADE,
    defect_category_id UUID NOT NULL REFERENCES qc_defect_categories(id) ON DELETE RESTRICT,
    component_id UUID REFERENCES components(id) ON DELETE SET NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    disposition VARCHAR(20) NOT NULL CHECK (disposition IN ('REPAIR', 'REJECT')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_qc_inspection_defects_inspection ON qc_inspection_defects(inspection_id);
CREATE INDEX IF NOT EXISTS idx_qc_inspection_defects_category ON qc_inspection_defects(defect_category_id);
CREATE INDEX IF NOT EXISTS idx_qc_inspection_defects_component ON qc_inspection_defects(component_id);

