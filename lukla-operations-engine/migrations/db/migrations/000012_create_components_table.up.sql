CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    code VARCHAR,
    name VARCHAR NOT NULL,
    sequence_number INT NOT NULL DEFAULT 1,
    standard_sam NUMERIC(6,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_components_article_id ON components (article_id);
CREATE INDEX IF NOT EXISTS idx_components_sequence ON components (article_id, sequence_number);

