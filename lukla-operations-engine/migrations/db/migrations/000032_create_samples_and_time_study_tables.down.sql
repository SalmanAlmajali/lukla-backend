-- Migration 000032: Create Samples and Time Study Tables (BPMN 03)
-- Down Migration

DROP INDEX IF EXISTS idx_orders_sample_version_id;
ALTER TABLE orders DROP COLUMN IF EXISTS sample_version_id;
ALTER TABLE orders DROP COLUMN IF EXISTS unit_price;
ALTER TABLE orders DROP COLUMN IF EXISTS total_amount;

DROP TABLE IF EXISTS sample_time_studies CASCADE;
DROP TABLE IF EXISTS sample_versions CASCADE;
DROP TABLE IF EXISTS sample_requests CASCADE;

