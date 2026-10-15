-- Migration 000031: Create Notifications Table and Add Completed At to Orders
-- Up Migration

-- 1. Tambah kolom completed_at ke tabel orders jika belum ada
ALTER TABLE orders ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

-- 2. Buat tabel notifications untuk in-app notification engine
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'ORDER_COMPLETED', 'MATERIAL_RECEIVED', 'PRODUCTION_MILESTONE', 'ANNOUNCEMENT'
    reference_id UUID,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexing untuk optimasi query portal klien
CREATE INDEX IF NOT EXISTS idx_notifications_client_id ON notifications(client_id);
CREATE INDEX IF NOT EXISTS idx_notifications_client_unread ON notifications(client_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

