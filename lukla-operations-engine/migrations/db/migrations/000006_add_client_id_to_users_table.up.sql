ALTER TABLE users
ADD COLUMN client_id UUID REFERENCES clients(id) ON DELETE SET NULL;

CREATE INDEX idx_users_client_id ON users (client_id);

