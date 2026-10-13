CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_code VARCHAR UNIQUE,
    full_name VARCHAR NOT NULL,
    email VARCHAR UNIQUE,
    phone_number VARCHAR,
    password_hash TEXT,
    role_id BIGINT NOT NULL REFERENCES roles(id),
    can_access_app BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_role_id ON users (role_id);