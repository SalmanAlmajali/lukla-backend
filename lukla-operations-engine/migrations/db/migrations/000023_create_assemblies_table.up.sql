-- Migration 000023: Create assemblies table (BPMN 05)
CREATE TABLE IF NOT EXISTS assemblies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE RESTRICT,
    quantity_assembled INT NOT NULL CHECK (quantity_assembled > 0),
    status VARCHAR(50) NOT NULL DEFAULT 'MENUNGGU_QC',
    assembled_date DATE NOT NULL DEFAULT CURRENT_DATE,
    assembled_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_assemblies_order_id ON assemblies(order_id);
CREATE INDEX IF NOT EXISTS idx_assemblies_article_id ON assemblies(article_id);
CREATE INDEX IF NOT EXISTS idx_assemblies_status ON assemblies(status);
CREATE INDEX IF NOT EXISTS idx_assemblies_assembled_date ON assemblies(assembled_date);

