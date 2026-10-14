CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE RESTRICT,
    target_quantity INTEGER NOT NULL CHECK (target_quantity > 0),
    status VARCHAR(30) NOT NULL DEFAULT 'IN_PRODUCTION', -- DRAFT, IN_PRODUCTION, COMPLETED, CANCELLED
    notes TEXT,
    start_date DATE,
    target_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders(client_id);
CREATE INDEX IF NOT EXISTS idx_orders_article_id ON orders(article_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

