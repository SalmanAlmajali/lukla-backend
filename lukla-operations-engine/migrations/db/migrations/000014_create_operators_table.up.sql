CREATE TABLE IF NOT EXISTS operators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(50),
    join_date DATE NOT NULL DEFAULT CURRENT_DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'ON_LEAVE')),
    default_line VARCHAR(50),
    skill_tier VARCHAR(50),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_operators_operator_code ON operators (operator_code);
CREATE INDEX IF NOT EXISTS idx_operators_status ON operators (status);
CREATE INDEX IF NOT EXISTS idx_operators_default_line ON operators (default_line);
CREATE INDEX IF NOT EXISTS idx_operators_user_id ON operators (user_id);

