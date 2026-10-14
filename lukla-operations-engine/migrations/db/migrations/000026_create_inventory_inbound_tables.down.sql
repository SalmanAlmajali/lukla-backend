-- Migration 000026: Down
DROP TABLE IF EXISTS inventory_receipt_items CASCADE;
DROP TABLE IF EXISTS inventory_lots CASCADE;
DROP TABLE IF EXISTS inventory_receipts CASCADE;
DROP TABLE IF EXISTS materials CASCADE;

