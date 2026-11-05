# SaaS E-Commerce & Multi-Tenant Management Platform

A production-grade multi-tenant e-commerce platform featuring a modern React/Vite dashboard frontend and a Go microservice backend with dynamic domain & subdomain management, PostgreSQL storage, and Caddy reverse proxy with On-Demand TLS.

---

## Architecture Overview

```
                      ┌──────────────────────────────────────────────┐
                      │             Incoming Traffic (:80, :443)      │
                      └──────────────────────┬───────────────────────┘
                                             │
                                             ▼
                      ┌──────────────────────────────────────────────┐
                      │              Caddy Reverse Proxy             │
                      │             (On-Demand TLS Hook)             │
                      └──────┬────────────────────────────────┬──────┘
                             │                                │
          Ask Endpoint for   │                                │ Proxy Request
          SSL Cert Issuance  │                                │ with Host Header
                             ▼                                ▼
                      ┌──────────────────────────────────────────────┐
                      │             Go Microservice (Chi)            │
                      │  - Caddy Ask: /api/v1/domains/validate       │
                      │  - Tenant Middleware (Host inspection)       │
                      │  - Store Management: /api/v1/stores          │
                      │  - Multi-Tenant Storefront Resolution        │
                      └──────────────────────┬───────────────────────┘
                                             │
                                             ▼
                      ┌──────────────────────────────────────────────┐
                      │              PostgreSQL 16 Database          │
                      │   - Stores, Subdomains, Custom Domains       │
                      │   - B-Tree Unique Lowercase Indexes          │
                      └──────────────────────────────────────────────┘
```

---

## Directory Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go              # Backend entrypoint with graceful shutdown
├── internal/
│   ├── config/                  # Environment variable configuration
│   ├── database/                # PostgreSQL connection pooling & migrations
│   ├── handler/                 # Store, domain validation, and storefront handlers
│   ├── middleware/              # Tenant resolution middleware via Host header
│   ├── model/                   # Data structures, DTOs & RFC 1123 validations
│   └── repository/              # SQL queries and case-insensitive resolution
├── migrations/
│   └── 000001_create_stores_table.sql # PostgreSQL schema and initial seeds
├── src/                         # Frontend React / Vite / Tailwind Dashboard
├── Caddyfile                    # Caddy reverse proxy and on_demand_tls configuration
├── Dockerfile                   # Multi-stage production Go Dockerfile
├── docker-compose.yml           # Compose setup for app, postgres, and caddy
├── go.mod                       # Go module dependencies
├── package.json                 # Frontend dependencies
└── README.md
```

---

## Quick Start (Docker Compose)

### 1. Start Services
```bash
docker compose up -d --build
```

### 2. Verify Containers
```bash
docker compose ps
```

Containers:
- `tenant_caddy`: Reverse proxy on ports `80` and `443`
- `tenant_app`: Go backend microservice on port `8080`
- `tenant_postgres`: PostgreSQL database on port `15432` (internal `5432`)

---

## API Endpoints

### Store Management
- `POST /api/v1/stores`: Create a new tenant store (`name`, `subdomain`, optional `custom_domain`)
- `GET /api/v1/stores`: List stores with pagination
- `GET /api/v1/stores/{id}`: Fetch store by UUID
- `POST /api/v1/stores/{id}/custom-domain`: Connect a custom domain
- `DELETE /api/v1/stores/{id}/custom-domain`: Disconnect custom domain

### Caddy On-Demand TLS
- `GET /api/v1/domains/validate?domain={domain}`: Checked by Caddy before issuing or renewing SSL certificates. Returns `200 OK` if valid and active, or `404`/`403` to reject unauthorized requests.

### Storefront Resolution
- `GET /`: Resolved dynamically by inspecting the `Host` header. Returns JSON or rendered storefront HTML corresponding to the active tenant store.

---

## Running the Frontend Dashboard

```bash
npm install
npm run dev
```

---

## Running Backend Tests

```bash
go test -v ./...
```

