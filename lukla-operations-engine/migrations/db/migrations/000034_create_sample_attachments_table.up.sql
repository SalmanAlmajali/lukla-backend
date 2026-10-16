-- Migration 000034: Create Sample Attachments Table (Amazon S3 Object Storage)

CREATE TABLE IF NOT EXISTS sample_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sample_request_id UUID NOT NULL REFERENCES sample_requests(id) ON DELETE CASCADE,
    sample_version_id UUID REFERENCES sample_versions(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    s3_bucket VARCHAR(255) NOT NULL,
    s3_key VARCHAR(500) NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    caption TEXT,
    attachment_type VARCHAR(50) NOT NULL DEFAULT 'PROTOTYPE_PHOTO',
    -- Values: 'PROTOTYPE_PHOTO', 'DEFECT_PHOTO', 'REVISION_REFERENCE', 'TECHPACK_SKETCH'
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    uploader_name VARCHAR(100) NOT NULL DEFAULT 'System',
    uploader_role VARCHAR(50) NOT NULL, -- 'CLIENT_PORTAL', 'SAMPLE_MAKER', 'RND_ENGINEER', 'FACTORY_MANAGER'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexing for high-performance version lookup and chronological ordering
CREATE INDEX IF NOT EXISTS idx_sample_attachments_request ON sample_attachments(sample_request_id);
CREATE INDEX IF NOT EXISTS idx_sample_attachments_version ON sample_attachments(sample_version_id);
CREATE INDEX IF NOT EXISTS idx_sample_attachments_created_at ON sample_attachments(created_at DESC);
