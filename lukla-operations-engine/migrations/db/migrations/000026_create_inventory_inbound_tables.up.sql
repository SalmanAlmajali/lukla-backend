-- Migration 000026: Create Inventory Inbound Tables (Materials, Receipts, Lots)
-- BPMN 02: Penerimaan Bahan & Blokade Kontaminasi Silang (FR-6.1 Mandatory Tagging)

CREATE TABLE IF NOT EXISTS materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    category VARCHAR(30) NOT NULL CHECK (category IN ('FABRIC', 'ACCESSORY', 'THREAD', 'PACKAGING', 'OTHER')),
    default_unit VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_materials_code ON materials(code);
CREATE INDEX IF NOT EXISTS idx_materials_category ON materials(category);

CREATE TABLE IF NOT EXISTS inventory_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_number VARCHAR(50) NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    received_date DATE NOT NULL,
    supplier_name VARCHAR(200),
    delivery_note_number VARCHAR(100),
    received_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_receipts_client ON inventory_receipts(client_id);
CREATE INDEX IF NOT EXISTS idx_inventory_receipts_order ON inventory_receipts(order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_receipts_date ON inventory_receipts(received_date);

CREATE TABLE IF NOT EXISTS inventory_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES inventory_receipts(id) ON DELETE CASCADE,
    lot_number VARCHAR(50) NOT NULL UNIQUE,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    initial_quantity NUMERIC(15, 2) NOT NULL CHECK (initial_quantity > 0),
    current_quantity NUMERIC(15, 2) NOT NULL CHECK (current_quantity >= 0),
    unit VARCHAR(20) NOT NULL,
    roll_count INT DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE' CHECK (status IN ('AVAILABLE', 'DEPLETED', 'RESERVED')),
    warehouse_location VARCHAR(100) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_lots_material ON inventory_lots(material_id);
CREATE INDEX IF NOT EXISTS idx_inventory_lots_client ON inventory_lots(client_id);
CREATE INDEX IF NOT EXISTS idx_inventory_lots_order ON inventory_lots(order_id);
CREATE INDEX IF NOT EXISTS idx_inventory_lots_status ON inventory_lots(status);
CREATE INDEX IF NOT EXISTS idx_inventory_lots_receipt ON inventory_lots(receipt_id);

CREATE TABLE IF NOT EXISTS inventory_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES inventory_receipts(id) ON DELETE CASCADE,
    lot_id UUID NOT NULL REFERENCES inventory_lots(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE RESTRICT,
    quantity NUMERIC(15, 2) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL,
    roll_count INT DEFAULT 1,
    warehouse_location VARCHAR(100) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_receipt_items_receipt ON inventory_receipt_items(receipt_id);
CREATE INDEX IF NOT EXISTS idx_inventory_receipt_items_lot ON inventory_receipt_items(lot_id);

-- Seed default initial materials for development and testing
INSERT INTO materials (code, name, category, default_unit, description)
VALUES 
    ('MAT-CTN-30S-BLK', 'Cotton Combed 30s Reaktif - Hitam Jet Black', 'FABRIC', 'KG', 'Kain katun combed rajut knits gramasi 140-150 gsm'),
    ('MAT-CTN-30S-WHT', 'Cotton Combed 30s Reaktif - Putih Solid', 'FABRIC', 'KG', 'Kain katun combed rajut knits putih solid'),
    ('MAT-RIB-30S-BLK', 'Rib Spandex 1x1 Cotton - Hitam', 'FABRIC', 'KG', 'Rib elastis untuk leher dan manset'),
    ('MAT-ZIP-YKK-05', 'Resleting YKK Metal Coil No. 5 Hitam 70cm', 'ACCESSORY', 'PCS', 'Ritsleting utama jaket dan hoodie'),
    ('MAT-THD-SPUN-BLK', 'Benang Jahit Spun Polyester 40/2 Hitam', 'THREAD', 'CONE', 'Benang jahit konveksi berkekuatan tinggi'),
    ('MAT-BTN-4H-BLK', 'Kancing Lubang 4 Resin 18L Hitam Doff', 'ACCESSORY', 'PCS', 'Kancing kemeja dan seragam 18L')
ON CONFLICT (code) DO NOTHING;

