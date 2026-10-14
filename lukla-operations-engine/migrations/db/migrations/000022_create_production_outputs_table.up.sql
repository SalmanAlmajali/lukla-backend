-- Production outputs table (BPMN 05: 30-minute interval output tracking by Mandor)
CREATE TABLE IF NOT EXISTS production_outputs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_id UUID NOT NULL REFERENCES lines(id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    time_slot VARCHAR(30) NOT NULL, -- e.g. "08:00-08:30", "08:30-09:00"
    work_date DATE NOT NULL DEFAULT CURRENT_DATE,
    snapshot_sewing_rate NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    notes TEXT,
    recorded_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_outputs_line_id ON production_outputs(line_id);
CREATE INDEX IF NOT EXISTS idx_production_outputs_order_id ON production_outputs(order_id);
CREATE INDEX IF NOT EXISTS idx_production_outputs_operator_id ON production_outputs(operator_id);
CREATE INDEX IF NOT EXISTS idx_production_outputs_component_id ON production_outputs(component_id);
CREATE INDEX IF NOT EXISTS idx_production_outputs_work_date ON production_outputs(work_date);
CREATE INDEX IF NOT EXISTS idx_production_outputs_time_slot ON production_outputs(time_slot);

