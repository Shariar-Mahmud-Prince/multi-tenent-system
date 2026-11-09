# Subdomain & Custom Domain Architecture Specification

**Component**: Centralized Domain Management System  
**Integration Points**: Go API, PostgreSQL, Caddy Reverse Proxy, Public DNS Resolvers  
**Compliance**: RFC 1035 / RFC 1123 DNS Standards

---

## 1. Domain Types Supported

### A. Platform Subdomains
- **Pattern**: `<subdomain>.<BASE_DOMAIN>` (e.g. `acme.yourplatform.com`)
- **Provisioning**: Automatic upon store creation.
- **TLS Coverage**: Covered automatically via On-Demand TLS or wildcard certificate.
- **Constraints**: 3–63 characters, lowercase alphanumeric and hyphens, not starting or ending with a hyphen, reserved system keywords blocked.

### B. Custom Domains
- **Pattern**: `customer-store.com` or `shop.customer-store.com`
- **Provisioning**: Managed via a strict 10-step DNS ownership verification workflow.
- **Supported DNS Records**:
  - `CNAME` record pointing to `<subdomain>.<BASE_DOMAIN>` (preferred for subdomains e.g. `shop.brand.com`).
  - `A` record pointing to the VPS public IP address (for root domains e.g. `brand.com`).
  - `TXT` verification record: `_saas-verify.brand.com` with value `token=<random_64_char_hex>`.

---

## 2. The 10-Step Domain Verification Workflow

```mermaid
sequenceDiagram
    autonumber
    actor Customer as Store Owner
    participant API as Go Backend API
    participant DB as PostgreSQL DB
    participant DNS as Public DNS System
    participant Caddy as Caddy Proxy
    participant ACME as Let's Encrypt / CA

    Customer->>API: 1. POST /api/v1/stores/{id}/custom-domain (domain.com)
    API->>API: 2. Validate FQDN format & conflict check
    API->>DB: 3. Generate verification token & save status='pending_verification'
    API-->>Customer: 4. Return DNS setup instructions (TXT & CNAME/A records)
    Customer->>DNS: 5. Configure DNS records at domain registrar
    Customer->>API: 6. POST /api/v1/stores/{id}/verify-domain
    API->>DNS: 7. Query TXT record & CNAME/A resolution
    alt DNS Matches Token & IP
        API->>DB: 8. Update status='verified' -> 'provisioning'
        Caddy->>API: 9. GET /api/v1/domains/validate?domain=domain.com
        API-->>Caddy: 200 OK (Domain is verified)
        Caddy->>ACME: Request TLS Certificate
        ACME-->>Caddy: Certificate Issued & Active
        API->>DB: 10. Update status='active'
        API-->>Customer: Domain Active & HTTPS Secure
    else DNS Records Missing or Incorrect
        API->>DB: Update status='failed' (retryable)
        API-->>Customer: Error: Required DNS records not detected yet
    end
```

---

## 3. Domain State Machine

```mermaid
stateDiagram-v2
    [*] --> pending_verification: Register Domain
    pending_verification --> verified: DNS TXT & CNAME Verified
    pending_verification --> failed: Verification Timed Out / Invalid
    failed --> pending_verification: Retry Verification
    verified --> provisioning: Initial TLS Request Triggered
    provisioning --> active: First Successful HTTPS Request
    active --> suspended: Store Disabled / Billing Lapsed
    suspended --> active: Store Reactivated
    active --> [*]: Domain Removed / Disconnected
```

| State | Caddy SSL Authorization | Storefront HTTP Access | Description |
| :--- | :--- | :--- | :--- |
| `pending_verification` | **DENIED (404)** | **BLOCKED (404)** | Domain entered by user, awaiting DNS token configuration. |
| `verified` | **ALLOWED (200)** | **STANDBY (200)** | DNS ownership confirmed. Ready for certificate issuance. |
| `provisioning` | **ALLOWED (200)** | **IN PROGRESS** | Caddy is actively obtaining the initial TLS certificate. |
| `active` | **ALLOWED (200)** | **LIVE (200)** | Fully live, secure HTTPS traffic served to customers. |
| `failed` | **DENIED (404)** | **BLOCKED (404)** | DNS verification failed or pointed to incorrect IP. |
| `suspended` | **DENIED (403)** | **BLOCKED (403)** | Store deactivated or subscription expired. |

---

## 4. Protection Against Domain Takeover

1. **Random Ownership Token**:
   - Every domain connection generates an unguessable 64-character cryptographic token:
     `saas-verify=a8f9c0e2b1d4...`
   - Stored in PostgreSQL with a 48-hour expiration.
2. **Prevent Duplicate Registrations**:
   - PostgreSQL unique index on `LOWER(custom_domain)` guarantees no two stores can claim the same domain name.
3. **Strict Validation Hook**:
   - `/api/v1/domains/validate` ONLY returns `200 OK` if:
     - The domain exists in the database.
     - The store is `is_active = true`.
     - The domain status is `verified` or `active`.
   - Never allows arbitrary hostnames to initiate certificate generation.
