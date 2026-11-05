-- Enable UUID extension if not already present
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create stores table
CREATE TABLE IF NOT EXISTS stores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    subdomain VARCHAR(63) NOT NULL,
    custom_domain VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Case-insensitive unique indexes for subdomains and custom domains
CREATE UNIQUE INDEX IF NOT EXISTS idx_stores_subdomain_unique_lower 
    ON stores (LOWER(subdomain));

CREATE UNIQUE INDEX IF NOT EXISTS idx_stores_custom_domain_unique_lower 
    ON stores (LOWER(custom_domain)) 
    WHERE custom_domain IS NOT NULL AND custom_domain <> '';

CREATE INDEX IF NOT EXISTS idx_stores_is_active 
    ON stores (is_active);

-- Seed initial test tenants for development and verification
INSERT INTO stores (id, name, subdomain, custom_domain, is_active)
VALUES 
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Acme Store', 'acme', 'acmeshop.local', TRUE),
    ('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'Starlight Coffee', 'starlight', 'starlightcafe.local', TRUE),
    ('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'Demo Boutique', 'demo', NULL, TRUE)
ON CONFLICT DO NOTHING;
