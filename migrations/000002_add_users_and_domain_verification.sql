-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'owner',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique_lower 
    ON users (LOWER(email));

-- Insert default admin user (password: Admin@123456)
-- Bcrypt hash: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
INSERT INTO users (id, email, password_hash, role)
VALUES (
    'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
    'admin@yourplatform.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'admin'
)
ON CONFLICT (id) DO NOTHING;

-- Alter stores table to add multi-tenant ownership and domain lifecycle
ALTER TABLE stores 
    ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS domain_status VARCHAR(50) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS verification_token VARCHAR(255),
    ADD COLUMN IF NOT EXISTS verification_expires_at TIMESTAMP WITH TIME ZONE;

-- Link existing stores to admin user
UPDATE stores SET owner_id = 'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a44' WHERE owner_id IS NULL;

-- Performance and tenant query indexes
CREATE INDEX IF NOT EXISTS idx_stores_owner_id ON stores (owner_id);
CREATE INDEX IF NOT EXISTS idx_stores_domain_status ON stores (domain_status);
