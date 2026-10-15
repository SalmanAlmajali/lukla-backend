-- Migration 000031: Create Notifications Table and Add Completed At to Orders
-- Down Migration

DROP TABLE IF EXISTS notifications CASCADE;
ALTER TABLE orders DROP COLUMN IF EXISTS completed_at;

