CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR NOT NULL UNIQUE,
    name VARCHAR NOT NULL,
    contact VARCHAR,
    address TEXT,
    client_type VARCHAR NOT NULL DEFAULT 'B2B',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_clients_code ON clients (code);

