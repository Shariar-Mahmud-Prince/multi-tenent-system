# Zero-Downtime Deployment & Database Migration Runbook

## 1. Principles
- Never restart database or Redis during rolling deployments.
- Database schema changes must remain backward-compatible (expand before contract).
- Caddy acts as a buffer with reverse proxy keepalive during app container swaps.

## 2. Standard Deployment Procedure

### Step 1: Run Pre-Flight Tests Locally
```bash
make test
```

### Step 2: Database Schema Migration
Migrations must be additive (e.g. adding columns with defaults or nullable):
```bash
docker exec -i tenant_postgres psql -U postgres -d tenant_domains < ./migrations/00000X_migration_name.sql
```

### Step 3: Zero-Downtime Application Container Rebuild
Rebuild the new container image and swap it with health-check verification:
```bash
# 1. Build new application image
docker compose build app

# 2. Recreate container gracefully
docker compose up -d --no-deps app

# 3. Verify health status before releasing traffic
docker inspect --format='{{json .State.Health.Status}}' tenant_app
```

### Step 4: Caddy Configuration Reload
If `Caddyfile` was modified, reload configuration without terminating active connections:
```bash
docker exec tenant_caddy caddy reload --config /etc/caddy/Caddyfile
```

### Step 5: Post-Deployment Smoke Verification
```bash
go run ./scripts/smoke/main.go
```
