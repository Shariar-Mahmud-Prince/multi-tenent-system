# Production Multi-Tenant SaaS Platform Foundation

A production-grade, white-label **Multi-Tenant SaaS Foundation Architecture** featuring a high-performance **Go backend**, a modular **React (Vite + TypeScript + Tailwind) frontend**, **PostgreSQL 16**, **Redis 7**, **Caddy v2 with On-Demand TLS**, and **Prometheus/Grafana observability**.

This architecture is **not tied to any specific vertical**; it is designed as a foundational engine to power **any multi-tenant SaaS application** (Learning Management Systems, CRM, Helpdesk, Portals, ERP, Project Management, E-Commerce, etc.) on a single VPS with horizontal expansion capability.

---

## Architecture Blueprint

```
                                 PUBLIC INTERNET
                                       │
                         [DNS: Subdomains & Custom Domains]
                                       │
                 ┌─────────────────────▼──────────────────────┐
                 │                Caddy v2                    │
                 │   - Automatic HTTPS / On-Demand TLS        │
                 │   - TLS Certificate Auto-Provisioning      │
                 │   - Security Headers (HSTS, CSP, nosniff)  │
                 │   - Ports 80 & 443 Public                  │
                 └──────────────┬──────────────────┬──────────┘
                                │                  │
                       TLS Ask  │                  │ Proxy
                    Validation  │                  │ Traffic
                                │                  │
                 ┌──────────────▼──────────────────▼──────────┐
                 │            Go Backend API                  │
                 │   - JWT Auth & Role-Based RBAC             │
                 │   - Hostname Tenant Resolver (346 ns/op)   │
                 │   - Cryptographic DNS Verifier             │
                 │   - Token-Bucket & Sliding Window Limiter  │
                 │   - Bot Traffic Filter & Security Guards   │
                 │   - Prometheus Observability (/metrics)    │
                 └──────────────┬──────────────────┬──────────┘
                                │                  │
                       Private  │                  │ Private
                      Container │                  │ Container
                                │                  │
                 ┌──────────────▼──────┐    ┌──────▼──────────┐
                 │    PostgreSQL 16    │    │     Redis 7     │
                 │  - Logical Tenancy  │    │  - Tenant Cache │
                 │  - Immutable UUIDs  │    │  - Rate Limits  │
                 │  - Domain Status    │    │  - Cache Flush  │
                 └─────────────────────┘    └─────────────────┘
```

---

## Core Capabilities

1. **Multi-Tenancy Out of the Box**:
   - Platform-provided subdomains: `workspace.yourdomain.com`.
   - Tenant-connected custom domains: `app.customer.com` with automated 10-step DNS TXT token verification.
   - Host-header tenant resolution via Redis cache and PostgreSQL in under 400 nanoseconds.
2. **On-Demand TLS with Strict Guard**:
   - Caddy provisions SSL certificates automatically from Let's Encrypt / ZeroSSL.
   - Guarded by backend authorization endpoint `/api/v1/domains/validate` to prevent certificate exhaustion attacks.
3. **Enterprise Security & IDOR Elimination**:
   - Bcrypt password hashing + signed JWT (HS256) auth.
   - Strict owner-scoped tenant mutations; cross-tenant updates/deletions blocked with HTTP 403.
   - Token-bucket rate limiting (50 req/s, burst 100) + hostile scanner tool blocking (`sqlmap`, `nikto`, `masscan`, etc.).
4. **Observability & Health**:
   - Prometheus metrics exposed via `/metrics` (request latency histograms, throughput, active tenant gauge).
   - Production alert rules configured (`prometheus/alert_rules.yml`).
   - Auto-provisioned Grafana dashboards.
5. **Operational Resilience**:
   - Automated compressed database backups with SHA256 checksums and 7-day retention rotation (`scripts/backup.sh`).
   - RPO < 1 hour, RTO < 15 minutes disaster recovery runbook.

---

## Complete Documentation Index

- [Master SaaS Setup & Architecture Guide](file:///P:/DashbordLMS/docs/SAAS_ARCHITECTURE_AND_SETUP_GUIDE.md)
- [Audit Report](file:///P:/DashbordLMS/docs/architecture/audit-report.md)
- [Production Readiness Plan](file:///P:/DashbordLMS/docs/architecture/production-readiness-plan.md)
- [System Architecture](file:///P:/DashbordLMS/docs/architecture/system-architecture.md)
- [Tenant Architecture](file:///P:/DashbordLMS/docs/architecture/tenant-architecture.md)
- [Domain Architecture](file:///P:/DashbordLMS/docs/architecture/domain-architecture.md)
- [Automatic HTTPS & TLS](file:///P:/DashbordLMS/docs/ssl/automatic-https.md)
- [Custom Domains & DNS Records](file:///P:/DashbordLMS/docs/ssl/custom-domains.md)
- [Certificate Troubleshooting](file:///P:/DashbordLMS/docs/ssl/certificate-troubleshooting.md)
- [Metrics, Alerts & Monitoring](file:///P:/DashbordLMS/docs/monitoring/metrics-and-alerts.md)
- [Disaster Recovery Runbook](file:///P:/DashbordLMS/docs/disaster-recovery/backup-and-restore.md)
- [Incident Response Runbook](file:///P:/DashbordLMS/docs/runbooks/incident-response.md)
- [Zero-Downtime Deployment](file:///P:/DashbordLMS/docs/runbooks/zero-downtime-deployment.md)
- [Final Test Report](file:///P:/DashbordLMS/docs/testing/final-test-report.md)

---

## Quick Start

### 1. Start Production Container Stack
```bash
make up
```

### 2. Run Test Matrix
```bash
make test
```

### 3. Run Live Smoke Check
```bash
go run ./scripts/smoke/main.go
```
