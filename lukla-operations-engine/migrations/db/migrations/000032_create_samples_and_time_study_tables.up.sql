-- Migration 000032: Create Samples and Time Study Tables (BPMN 03)
-- Up Migration

-- 1. Tabel Permintaan Sample (sample_requests)
CREATE TABLE IF NOT EXISTS sample_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_code VARCHAR(50) NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE RESTRICT,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- DRAFT, IN_DEVELOPMENT, IN_REVIEW, REVISED, APPROVED, REJECTED, CONVERTED_TO_ORDER
    target_date DATE,
    initial_notes TEXT,
    converted_order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexing sample_requests
CREATE INDEX IF NOT EXISTS idx_sample_requests_client ON sample_requests(client_id);
CREATE INDEX IF NOT EXISTS idx_sample_requests_article ON sample_requests(article_id);
CREATE INDEX IF NOT EXISTS idx_sample_requests_status ON sample_requests(status);
CREATE INDEX IF NOT EXISTS idx_sample_requests_created_at ON sample_requests(created_at DESC);

-- 2. Tabel Versi Sample (sample_versions) - Versioning Engine (v1, v2, dst)
CREATE TABLE IF NOT EXISTS sample_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_request_id UUID NOT NULL REFERENCES sample_requests(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    version_label VARCHAR(20) NOT NULL, -- 'v1', 'v2', dst
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- DRAFT, IN_DEVELOPMENT, IN_REVIEW, REVISED, APPROVED, REJECTED
    revision_notes TEXT,
    client_feedback TEXT,
    total_sam NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    proposed_unit_price NUMERIC(15,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    price_notes TEXT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sample_versions_request_num UNIQUE (sample_request_id, version_number)
);

-- Indexing sample_versions
CREATE INDEX IF NOT EXISTS idx_sample_versions_request ON sample_versions(sample_request_id);
CREATE INDEX IF NOT EXISTS idx_sample_versions_status ON sample_versions(status);
CREATE INDEX IF NOT EXISTS idx_sample_versions_created_at ON sample_versions(created_at DESC);

-- 3. Tabel Observasi Waktu Kerja Stopwatch (sample_time_studies) per Komponen
CREATE TABLE IF NOT EXISTS sample_time_studies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_version_id UUID NOT NULL REFERENCES sample_versions(id) ON DELETE CASCADE,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    observed_seconds NUMERIC(10,2) NOT NULL CHECK (observed_seconds > 0),
    allowance_percentage NUMERIC(5,2) NOT NULL DEFAULT 15.00 CHECK (allowance_percentage >= 0),
    calculated_sam NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    target_per_hour INT NOT NULL DEFAULT 0,
    operator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sample_time_studies_version_comp UNIQUE (sample_version_id, component_id)
);

-- Indexing sample_time_studies
CREATE INDEX IF NOT EXISTS idx_sample_time_studies_version ON sample_time_studies(sample_version_id);
CREATE INDEX IF NOT EXISTS idx_sample_time_studies_component ON sample_time_studies(component_id);

-- 4. Dukungan Tambahan pada Tabel orders untuk Konversi Sample dan Commercial Rate Card
ALTER TABLE orders ADD COLUMN IF NOT EXISTS sample_version_id UUID REFERENCES sample_versions(id) ON DELETE SET NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS unit_price NUMERIC(15,2);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_amount NUMERIC(15,2);

CREATE INDEX IF NOT EXISTS idx_orders_sample_version_id ON orders(sample_version_id);

