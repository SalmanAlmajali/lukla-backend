-- Production lines table
CREATE TABLE IF NOT EXISTS lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    max_capacity INTEGER NOT NULL DEFAULT 20 CHECK (max_capacity > 0),
    supervisor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, INACTIVE, MAINTENANCE
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lines_code ON lines(code);
CREATE INDEX IF NOT EXISTS idx_lines_supervisor_id ON lines(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_lines_status ON lines(status);

-- Line allocations table (Operator assignment to line & order)
CREATE TABLE IF NOT EXISTS line_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_id UUID NOT NULL REFERENCES lines(id) ON DELETE RESTRICT,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    assigned_component_id UUID REFERENCES components(id) ON DELETE SET NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, COMPLETED, TRANSFERRED
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_line_allocations_line_id ON line_allocations(line_id);
CREATE INDEX IF NOT EXISTS idx_line_allocations_order_id ON line_allocations(order_id);
CREATE INDEX IF NOT EXISTS idx_line_allocations_operator_id ON line_allocations(operator_id);
CREATE INDEX IF NOT EXISTS idx_line_allocations_status ON line_allocations(status);

-- Line transfers table (BPMN 04: audit log for operator line movement)
CREATE TABLE IF NOT EXISTS line_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    allocation_id UUID NOT NULL REFERENCES line_allocations(id) ON DELETE CASCADE,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    from_line_id UUID NOT NULL REFERENCES lines(id) ON DELETE RESTRICT,
    to_line_id UUID NOT NULL REFERENCES lines(id) ON DELETE RESTRICT,
    reason TEXT NOT NULL,
    transferred_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_line_transfers_allocation_id ON line_transfers(allocation_id);
CREATE INDEX IF NOT EXISTS idx_line_transfers_operator_id ON line_transfers(operator_id);
CREATE INDEX IF NOT EXISTS idx_line_transfers_from_line_id ON line_transfers(from_line_id);
CREATE INDEX IF NOT EXISTS idx_line_transfers_to_line_id ON line_transfers(to_line_id);

