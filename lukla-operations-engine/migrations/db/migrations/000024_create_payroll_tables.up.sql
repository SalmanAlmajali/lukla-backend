-- 000024_create_payroll_tables.up.sql
-- Tahap 2: Payroll Mingguan (BPMN 07: Borongan, Uang Makan & Penguncian)

CREATE TABLE IF NOT EXISTS payroll_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_code VARCHAR(50) NOT NULL UNIQUE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    meal_allowance_percentage NUMERIC(5, 2) NOT NULL CHECK (meal_allowance_percentage >= 10.00 AND meal_allowance_percentage <= 15.00),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'CALCULATED', 'LOCKED')),
    total_operators_count INT NOT NULL DEFAULT 0,
    total_gross_piece_rate NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_meal_allowance NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_adjustments NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    total_net_payable NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    locked_at TIMESTAMPTZ,
    locked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payroll_periods_dates ON payroll_periods(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_payroll_periods_status ON payroll_periods(status);
CREATE INDEX IF NOT EXISTS idx_payroll_periods_code ON payroll_periods(period_code);

CREATE TABLE IF NOT EXISTS payroll_slips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    total_pieces INT NOT NULL DEFAULT 0,
    gross_piece_rate_total NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    meal_allowance_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    adjustments_total NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    net_payable_total NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'CALCULATED' CHECK (status IN ('CALCULATED', 'LOCKED', 'PAID')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payroll_slips_period_operator UNIQUE (payroll_period_id, operator_id)
);

CREATE INDEX IF NOT EXISTS idx_payroll_slips_period_id ON payroll_slips(payroll_period_id);
CREATE INDEX IF NOT EXISTS idx_payroll_slips_operator_id ON payroll_slips(operator_id);
CREATE INDEX IF NOT EXISTS idx_payroll_slips_status ON payroll_slips(status);

CREATE TABLE IF NOT EXISTS payroll_slip_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_slip_id UUID NOT NULL REFERENCES payroll_slips(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE RESTRICT,
    work_date DATE NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    snapshot_sewing_rate NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    subtotal NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payroll_slip_items_slip_id ON payroll_slip_items(payroll_slip_id);
CREATE INDEX IF NOT EXISTS idx_payroll_slip_items_order_id ON payroll_slip_items(order_id);
CREATE INDEX IF NOT EXISTS idx_payroll_slip_items_component_id ON payroll_slip_items(component_id);
CREATE INDEX IF NOT EXISTS idx_payroll_slip_items_work_date ON payroll_slip_items(work_date);

