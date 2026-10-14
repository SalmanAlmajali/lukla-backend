-- Migration 000027: Seed Inventory Inbound Receipts & Lots
-- Skenario BPMN 02: Stok Bahan Bertagging Snøhund (PO-2026-10-001) & Aventura (PO-2026-10-002)

DO $$
DECLARE
    v_warehouse_user_id UUID;
    v_client_snohund_id UUID;
    v_client_aventura_id UUID;
    v_order_snohund_id UUID;
    v_order_aventura_id UUID;
    v_mat_ctn_blk_id UUID;
    v_mat_ctn_wht_id UUID;
    v_receipt_snohund_id UUID;
    v_receipt_aventura_id UUID;
    v_lot_snohund_1_id UUID;
    v_lot_snohund_2_id UUID;
    v_lot_aventura_1_id UUID;
BEGIN
    -- 1. Ambil ID relasi
    SELECT id INTO v_warehouse_user_id FROM users WHERE email = 'warehouse@lukla.local' LIMIT 1;
    SELECT id INTO v_client_snohund_id FROM clients WHERE code = 'SNOHUND' LIMIT 1;
    SELECT id INTO v_client_aventura_id FROM clients WHERE code = 'AVENTURA' LIMIT 1;
    SELECT id INTO v_order_snohund_id FROM orders WHERE order_number = 'PO-2026-10-001' LIMIT 1;
    SELECT id INTO v_order_aventura_id FROM orders WHERE order_number = 'PO-2026-10-002' LIMIT 1;
    SELECT id INTO v_mat_ctn_blk_id FROM materials WHERE code = 'MAT-CTN-30S-BLK' LIMIT 1;
    SELECT id INTO v_mat_ctn_wht_id FROM materials WHERE code = 'MAT-CTN-30S-WHT' LIMIT 1;

    -- Pastikan semua relasi ada sebelum seeding
    IF v_warehouse_user_id IS NOT NULL AND v_client_snohund_id IS NOT NULL AND v_order_snohund_id IS NOT NULL AND v_mat_ctn_blk_id IS NOT NULL THEN
        -- 2. Seed Receipt Snøhund
        INSERT INTO inventory_receipts (
            receipt_number,
            client_id,
            order_id,
            received_date,
            supplier_name,
            delivery_note_number,
            received_by_user_id,
            notes
        ) VALUES (
            'RCV-20261014-0001',
            v_client_snohund_id,
            v_order_snohund_id,
            '2026-10-14',
            'PT Indo Tekstil Bandung',
            'SJ-ITB-2026-8801',
            v_warehouse_user_id,
            'Penerimaan bahan kain utama Cotton Combed 30s Hitam untuk PO Snøhund'
        )
        ON CONFLICT (receipt_number) DO NOTHING
        RETURNING id INTO v_receipt_snohund_id;

        IF v_receipt_snohund_id IS NULL THEN
            SELECT id INTO v_receipt_snohund_id FROM inventory_receipts WHERE receipt_number = 'RCV-20261014-0001';
        END IF;

        -- 3. Seed Lots Snøhund
        INSERT INTO inventory_lots (
            receipt_id,
            lot_number,
            material_id,
            client_id,
            order_id,
            initial_quantity,
            current_quantity,
            unit,
            roll_count,
            status,
            warehouse_location,
            notes
        ) VALUES (
            v_receipt_snohund_id,
            'LOT-SNO-20261014-01',
            v_mat_ctn_blk_id,
            v_client_snohund_id,
            v_order_snohund_id,
            150.50,
            150.50,
            'KG',
            1,
            'AVAILABLE',
            'GUDANG-A-01',
            'Roll 1 kain Cotton Combed 30s Hitam kondisi prima'
        )
        ON CONFLICT (lot_number) DO NOTHING
        RETURNING id INTO v_lot_snohund_1_id;

        IF v_lot_snohund_1_id IS NULL THEN
            SELECT id INTO v_lot_snohund_1_id FROM inventory_lots WHERE lot_number = 'LOT-SNO-20261014-01';
        END IF;

        INSERT INTO inventory_lots (
            receipt_id,
            lot_number,
            material_id,
            client_id,
            order_id,
            initial_quantity,
            current_quantity,
            unit,
            roll_count,
            status,
            warehouse_location,
            notes
        ) VALUES (
            v_receipt_snohund_id,
            'LOT-SNO-20261014-02',
            v_mat_ctn_blk_id,
            v_client_snohund_id,
            v_order_snohund_id,
            149.50,
            149.50,
            'KG',
            1,
            'AVAILABLE',
            'GUDANG-A-02',
            'Roll 2 kain Cotton Combed 30s Hitam kondisi prima'
        )
        ON CONFLICT (lot_number) DO NOTHING
        RETURNING id INTO v_lot_snohund_2_id;

        IF v_lot_snohund_2_id IS NULL THEN
            SELECT id INTO v_lot_snohund_2_id FROM inventory_lots WHERE lot_number = 'LOT-SNO-20261014-02';
        END IF;

        -- 4. Seed Receipt Items Snøhund
        INSERT INTO inventory_receipt_items (
            receipt_id,
            lot_id,
            material_id,
            quantity,
            unit,
            roll_count,
            warehouse_location,
            notes
        ) VALUES 
            (v_receipt_snohund_id, v_lot_snohund_1_id, v_mat_ctn_blk_id, 150.50, 'KG', 1, 'GUDANG-A-01', 'Roll 1 kondisi prima'),
            (v_receipt_snohund_id, v_lot_snohund_2_id, v_mat_ctn_blk_id, 149.50, 'KG', 1, 'GUDANG-A-02', 'Roll 2 kondisi prima')
        ON CONFLICT DO NOTHING;
    END IF;

    -- 5. Seed Receipt Aventura (jika ada order PO-2026-10-002)
    IF v_warehouse_user_id IS NOT NULL AND v_client_aventura_id IS NOT NULL AND v_order_aventura_id IS NOT NULL AND v_mat_ctn_wht_id IS NOT NULL THEN
        INSERT INTO inventory_receipts (
            receipt_number,
            client_id,
            order_id,
            received_date,
            supplier_name,
            delivery_note_number,
            received_by_user_id,
            notes
        ) VALUES (
            'RCV-20261014-0002',
            v_client_aventura_id,
            v_order_aventura_id,
            '2026-10-14',
            'CV Rajut Makmur Cimahi',
            'SJ-RMC-2026-1044',
            v_warehouse_user_id,
            'Penerimaan bahan kain Cotton Combed 30s Putih untuk PO Aventura'
        )
        ON CONFLICT (receipt_number) DO NOTHING
        RETURNING id INTO v_receipt_aventura_id;

        IF v_receipt_aventura_id IS NULL THEN
            SELECT id INTO v_receipt_aventura_id FROM inventory_receipts WHERE receipt_number = 'RCV-20261014-0002';
        END IF;

        INSERT INTO inventory_lots (
            receipt_id,
            lot_number,
            material_id,
            client_id,
            order_id,
            initial_quantity,
            current_quantity,
            unit,
            roll_count,
            status,
            warehouse_location,
            notes
        ) VALUES (
            v_receipt_aventura_id,
            'LOT-AVE-20261014-01',
            v_mat_ctn_wht_id,
            v_client_aventura_id,
            v_order_aventura_id,
            100.00,
            100.00,
            'KG',
            1,
            'AVAILABLE',
            'GUDANG-B-01',
            'Roll 1 kain Cotton Combed 30s Putih kondisi bersih'
        )
        ON CONFLICT (lot_number) DO NOTHING
        RETURNING id INTO v_lot_aventura_1_id;

        IF v_lot_aventura_1_id IS NULL THEN
            SELECT id INTO v_lot_aventura_1_id FROM inventory_lots WHERE lot_number = 'LOT-AVE-20261014-01';
        END IF;

        INSERT INTO inventory_receipt_items (
            receipt_id,
            lot_id,
            material_id,
            quantity,
            unit,
            roll_count,
            warehouse_location,
            notes
        ) VALUES 
            (v_receipt_aventura_id, v_lot_aventura_1_id, v_mat_ctn_wht_id, 100.00, 'KG', 1, 'GUDANG-B-01', 'Roll 1 kondisi bersih')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

