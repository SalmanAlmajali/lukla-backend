CREATE TABLE IF NOT EXISTS rate_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE RESTRICT,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    sewing_rate NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    cutting_rate NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_rate_cards_component UNIQUE (article_id, component_id)
);

CREATE INDEX IF NOT EXISTS idx_rate_cards_client_id ON rate_cards(client_id);
CREATE INDEX IF NOT EXISTS idx_rate_cards_article_id ON rate_cards(article_id);
CREATE INDEX IF NOT EXISTS idx_rate_cards_component_id ON rate_cards(component_id);

CREATE TABLE IF NOT EXISTS rate_card_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_card_id UUID NOT NULL REFERENCES rate_cards(id) ON DELETE CASCADE,
    old_sewing_rate NUMERIC(12, 2) NOT NULL,
    new_sewing_rate NUMERIC(12, 2) NOT NULL,
    old_cutting_rate NUMERIC(12, 2) NOT NULL,
    new_cutting_rate NUMERIC(12, 2) NOT NULL,
    changed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rate_card_histories_rate_card_id ON rate_card_histories(rate_card_id);

