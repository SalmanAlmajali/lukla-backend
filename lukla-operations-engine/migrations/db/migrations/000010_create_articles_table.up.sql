CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    code VARCHAR NOT NULL UNIQUE,
    title VARCHAR NOT NULL,
    category VARCHAR NOT NULL CHECK (category IN ('GARMENT', 'BAG')),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_articles_client_id ON articles (client_id);
CREATE INDEX idx_articles_code ON articles (code);
CREATE INDEX idx_articles_category ON articles (category);

