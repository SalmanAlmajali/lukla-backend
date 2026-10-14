-- Migration 000027 Down: Rollback Seed Inventory Inbound Receipts & Lots

DELETE FROM inventory_receipts 
WHERE receipt_number IN ('RCV-20261014-0001', 'RCV-20261014-0002');

