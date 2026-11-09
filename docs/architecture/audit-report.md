# Comprehensive Codebase & Architecture Audit Report

**Date**: November 2026  
**Auditor**: Principal Software Architect, Senior Security & SRE Engineer  
**Scope**: Full stack (Go API, React Frontend, PostgreSQL, Caddy Reverse Proxy, Docker Compose)  
**Target Environment**: Single VPS Initial Deployment $\rightarrow$ Multi-Server Ready

---

## 1. Existing Architecture Overview

The system currently comprises:
- **Reverse Proxy**: Caddy v2 configured with On-Demand TLS (`ask http://app:8080/api/v1/domains/validate`). Caddy terminates SSL/TLS on `:443` and handles HTTP on `:80`.
- **Backend API**: Go 1.24 microservice using `chi` router, native `database/sql` connection pooling, and `github.com/lib/pq` PostgreSQL driver.
- **Frontend Dashboard**: React 19 + Vite 6 + Tailwind CSS + Radix UI component library (`DashbordLMS`), currently driven by static mock data (`src/services/mockData.ts`).
- **Database**: PostgreSQL 16 Alpine with initial schema migration (`stores` table) storing store names, subdomains, custom domains, and activation flags.
- **Traffic Protection**: In-memory token bucket rate limiter (50 req/sec, burst 100) and regex-based User-Agent security scanner filter (`internal/middleware/security.go`).
- **Orchestration**: Docker Compose running 3 services: `tenant_postgres`, `tenant_app`, and `tenant_caddy`.

```
Clients (Browser, Mobile, API)
             │
             ▼
   Caddy Reverse Proxy (:80, :443)
             │
             ├── On-Demand TLS Check ──► /api/v1/domains/validate
             └── Reverse Proxy (Host) ──► Go Microservice (Port 8080)
                                                 │
                                                 ▼
                                        PostgreSQL 16 (Port 5432)
```

---

## 2. Implemented Features

| Feature | Implementation Location | Assessment |
| :--- | :--- | :--- |
| **Subdomain Validation** | [`internal/model/store.go`](file:///p:/DashbordLMS/internal/model/store.go) | Compliant with RFC 1123 label specifications (3-63 chars, alphanumeric, hyphens allowed, no starting/ending hyphens). Reserved platform keywords (`admin`, `api`, `app`, `caddy`, `www`, etc.) are blocked. |
| **Custom Domain Validation** | [`internal/model/store.go`](file:///p:/DashbordLMS/internal/model/store.go) | Validates FQDN format; prevents tenants from claiming the root platform domain. |
| **Store Management CRUD** | [`internal/handler/store_handler.go`](file:///p:/DashbordLMS/internal/handler/store_handler.go) | Endpoints to create stores, list stores (paginated), retrieve store by ID, and connect/disconnect custom domains. |
| **On-Demand TLS Validation** | [`internal/handler/domain_handler.go`](file:///p:/DashbordLMS/internal/handler/domain_handler.go) | Caddy ask endpoint returning `200 OK` for authorized active tenant domains and system hosts; returns `404` or `403` for unknown or inactive domains. |
| **Dynamic Host Routing** | [`internal/middleware/tenant.go`](file:///p:/DashbordLMS/internal/middleware/tenant.go) | Inspects incoming `Host` header, queries PostgreSQL, injects `*model.Store` into context, and routes to storefront handler. |
| **Storefront Rendering** | [`internal/handler/storefront_handler.go`](file:///p:/DashbordLMS/internal/handler/storefront_handler.go) | Serves responsive HTML or structured JSON based on `Accept` header. |
| **Security & Rate Limiting** | [`internal/middleware/security.go`](file:///p:/DashbordLMS/internal/middleware/security.go) | Token bucket IP rate limiting + malicious scanner User-Agent blocking (`sqlmap`, `nikto`, `masscan`, etc.). |

---

## 3. Missing Features (Production Launch Gaps)

1. **Authentication & Identity Management**:
   - There is currently **no user registration or login**.
   - No JWT or session authentication system.
   - Passwords and credentials management do not exist yet.
2. **Tenant Ownership & Authorization (RBAC)**:
   - Stores are not linked to an owning user account or tenant ID.
   - Any client that knows a store UUID can modify or disconnect its custom domain.
3. **Custom Domain Ownership Verification Workflow**:
   - Currently, a custom domain is attached and activated **immediately** upon request.
   - Missing the required 10-step domain verification state machine (`pending_verification` $\rightarrow$ verification token generation $\rightarrow$ DNS TXT/CNAME validation $\rightarrow$ `verified` $\rightarrow$ `provisioning` $\rightarrow$ `active` $\rightarrow$ `failed`).
4. **Redis Integration**:
   - Redis is not yet provisioned in `docker-compose.yml`.
   - Rate limiting is purely node-local in-memory; not shared across multiple API replicas.
   - No caching of resolved tenant stores or domain lookups.
5. **Observability & Monitoring**:
   - Prometheus, Grafana, Node Exporter, cAdvisor, and PostgreSQL Exporter are absent from infrastructure.
   - Application-level Prometheus metrics (`/metrics`) are not exposed.
6. **Frontend - Backend API Integration**:
   - The React dashboard (`DashbordLMS`) displays mock data (`src/services/mockData.ts`) rather than communicating with the live Go backend API.
7. **Automated Database Backups & Disaster Recovery**:
   - No scheduled `pg_dump` backup cron or container.
   - No offsite storage sync, backup retention policy, or integrity check.
8. **Organized Top-Level `tests/` Directory**:
   - Tests are scattered inside `internal/` packages; no structured `tests/unit`, `tests/integration`, `tests/e2e`, `tests/performance`, `tests/security`, `tests/deployment`, `tests/resource`.
9. **CI/CD Pipeline**:
   - Missing automated GitHub Actions pipeline for linting, test suites, vulnerability scanning, and Docker builds.
10. **Makefile**:
    - Centralized `make` commands for developer operations and CI/CD are missing.

---

## 4. Existing Bugs & Code Issues

1. **Host Header Port Extraction Edge Case**:
   - In `TenantMiddleware`, if a client sends an IPv6 host with port (e.g. `[::1]:8080`), naive splitting can fail if not properly handled by `net.SplitHostPort`.
2. **Database Migration Error Swallowing**:
   - If a migration file is partially executed or fails mid-statement, non-transactional DDL in PostgreSQL can leave the schema in an inconsistent state.
3. **Hardcoded Internal Base Domain in Tests**:
   - Test files reference `"yourplatform.test"` while runtime defaults to `"yourplatform.local"`.

---

## 5. Security Vulnerabilities

| Vulnerability | Severity | Impact | Mitigation Plan |
| :--- | :--- | :--- | :--- |
| **Insecure Direct Object Reference (IDOR)** | **CRITICAL** | `POST /api/v1/stores/{id}/custom-domain` has no authentication check. Any attacker can hijack or alter domains of any store. | Implement JWT Authentication + Tenant Ownership Authorization middleware. |
| **Unverified Domain Registration & SSL Hijacking** | **HIGH** | An attacker could register `victim.com` or `google.com` as their custom domain; Caddy would then attempt SSL validation. | Enforce DNS TXT/CNAME token verification before marking domain `verified` and `active`. |
| **Public Database Exposure** | **HIGH** | `docker-compose.yml` maps `15432:5432` to `0.0.0.0`. In production, PostgreSQL must NEVER be exposed publicly. | Restrict PostgreSQL to internal Docker bridge network only. |
| **Missing Security Headers** | **MEDIUM** | Missing `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Strict-Transport-Security`. | Add standardized Security Headers middleware in Go and Caddy. |
| **CORS Unrestricted** | **MEDIUM** | API lacks explicit CORS middleware, exposing authenticated endpoints to cross-origin abuse. | Implement strict CORS middleware allowing only authorized origins. |

---

## 6. Tenant Isolation Risks

1. **Lack of Tenant ID Context in Store Schema**:
   - The `stores` table lacks an `owner_id` or `tenant_id` foreign key.
   - All tenant queries are currently scoped by `subdomain` or `custom_domain`, but not by authenticated user identity.
2. **Cache Key Collision (Future Redis Risk)**:
   - When Redis caching is introduced, failing to namespace keys with `tenant:{tenant_id}:...` will cause cross-tenant cache leaks.
3. **Row-Level Security (RLS)**:
   - PostgreSQL RLS is not currently enabled. While application-level tenancy will be enforced in Go repositories, RLS policies will provide defense-in-depth.

---

## 7. Deployment Risks (Single VPS Target)

1. **Resource Starvation (OOM Killer)**:
   - `docker-compose.yml` currently specifies no memory limits (`mem_limit`) or CPU reservations. A high-traffic spike or memory leak could crash the entire VPS.
2. **Single Point of Failure (SPOF)**:
   - Database, Redis, Caddy, and App all run on one VPS. Without automated daily offsite backups, hardware or disk failure causes total business data loss.
3. **Port 8080 Publicly Accessible**:
   - Currently, `tenant_app` publishes `8080:8080`. External clients could bypass Caddy and its rate-limiting/TLS layers entirely.

---

## 8. Performance Problems

1. **Direct Database Query on Every Request**:
   - Every single storefront hit executes a PostgreSQL `SELECT` query in `ResolveStoreByHost`.
   - Under heavy traffic (e.g. 5,000 req/sec), database connection pool exhaustion will occur.
2. **Absence of Cache Tier**:
   - Store metadata (ID, Subdomain, Custom Domain, IsActive) changes infrequently and should be cached in Redis with TTL and cache invalidation on update.

---

## 9. Testing Gaps

1. **No Integration Tests with Real Containers**:
   - Current tests use in-memory mock repositories (`mockStoreRepository`). Real PostgreSQL transactions, constraint violations, and index behavior are untested.
2. **No E2E Customer Journey Tests**:
   - Registration $\rightarrow$ Login $\rightarrow$ Store Creation $\rightarrow$ Domain Verification $\rightarrow$ Storefront Visit workflow is not verified end-to-end.
3. **No Automated Security Scanning**:
   - `govulncheck`, `trivy`, and OWASP ZAP are not wired into the test pipeline.

---

## 10. Recommended Improvements & Action Plan

```
PHASE 1: Complete codebase audit. (COMPLETED)
PHASE 2: Production architecture & implementation plan.
PHASE 3: Dedicated test infrastructure (tests/ directory layout).
PHASE 4: Unit and integration testing (Testcontainers for Go).
PHASE 5: Tenant isolation, authentication, and authorization (JWT + Tenant RBAC).
PHASE 6: Subdomain & custom domain management (DNS token verification workflow).
PHASE 7: Redis integration (caching, distributed rate limiting, traffic protection).
PHASE 8: Caddy, HTTPS, Docker, and deployment improvements (single VPS hardening).
PHASE 9: Prometheus, Grafana, and monitoring stack.
PHASE 10: k6 performance, stress, spike, and CPU/RAM benchmarking.
PHASE 11: Database backup & disaster recovery (pg_dump, retention, restore test).
PHASE 12: CI/CD pipeline & complete documentation under docs/.
PHASE 13: Final production-readiness verification.
```
