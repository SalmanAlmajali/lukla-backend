-- 000025_create_payroll_adjustments_table.up.sql
-- Tahap 2.2: Payroll Adjustments & Audit Trail (BPMN 07)

CREATE TABLE IF NOT EXISTS payroll_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    operator_id UUID NOT NULL REFERENCES operators(id) ON DELETE RESTRICT,
    adjustment_type VARCHAR(20) NOT NULL CHECK (adjustment_type IN ('BONUS', 'DEDUCTION', 'CORRECTION', 'REVERSAL')),
    amount NUMERIC(12, 2) NOT NULL,
    reason TEXT NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payroll_adjustments_period_id ON payroll_adjustments(payroll_period_id);
CREATE INDEX IF NOT EXISTS idx_payroll_adjustments_operator_id ON payroll_adjustments(operator_id);
CREATE INDEX IF NOT EXISTS idx_payroll_adjustments_created_at ON payroll_adjustments(created_at);

