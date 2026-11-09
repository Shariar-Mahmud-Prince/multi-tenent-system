# Caddy Automatic HTTPS & On-Demand TLS Architecture

## 1. Overview
In a multi-tenant SaaS platform where end customers can dynamically connect arbitrary custom domains (e.g. `shop.merchant.com`) and subdomains (e.g. `merchant.platform.com`), traditional static SSL certificates are insufficient.

The platform utilizes **Caddy v2** as the edge reverse proxy, leveraging its native **On-Demand TLS** capability. When a TLS handshake arrives for an unfamiliar hostname, Caddy halts the handshake, contacts our Go API authorization endpoint (`/api/v1/domains/validate?domain=<hostname>`), and obtains an automatic certificate from Let's Encrypt or ZeroSSL in real-time only if authorized.

## 2. On-Demand TLS Architecture & Request Flow
```
Client (Browser)                 Caddy Proxy                Go API Backend           ACME CA (Let's Encrypt)
      │                               │                           │                           │
      ├─── TLS ClientHello ──────────>│                           │                           │
      │    (SNI: shop.client.com)     │                           │                           │
      │                               ├─── GET /api/v1/domains/   │                           │
      │                               │    validate?domain=... ──>│                           │
      │                               │                           │                           │
      │                               │<── HTTP 200 OK (Allowed) ─┤                           │
      │                               │                           │                           │
      │                               ├─── ACME HTTP-01 / TLS-ALPN Challenge ────────────────>│
      │                               │<── Signed TLS Certificate ────────────────────────────┤
      │<── TLS Handshake Complete ────┤                           │                           │
      │                               │                           │                           │
      ├─── HTTPS GET / ──────────────>├─── Reverse Proxy (app) ──>│                           │
```

## 3. The Critical Security Safeguard: The `ask` Endpoint
Without strict authorization, On-Demand TLS introduces a severe **Denial of Service (DoS)** vulnerability: an attacker could direct thousands of wildcard domains to our VPS IP, exhausting Let's Encrypt rate limits and disk space for certificates.

To prevent this:
1. Caddy's `on_demand_tls` directive calls our internal authorization hook:
   ```caddy
   on_demand_tls {
       ask http://app:8080/api/v1/domains/validate
   }
   ```
2. The Go backend enforces:
   - System domains (`yourplatform.local`, `api.*`) are approved immediately.
   - Platform subdomains (`<subdomain>.yourplatform.local`) are approved only if the store exists and is `is_active = true`.
   - Custom domains (`shop.merchant.com`) are approved **ONLY** if the store exists, is active, AND has reached `domain_status = 'verified'` or `'active'`.
   - Domains in `pending_verification` or unassigned domains are rejected with `403 Forbidden` or `404 Not Found`.

## 4. Local Development vs. Production
- **Local Development**: In `Caddyfile`, `issuer internal` allows Caddy to act as a self-signed local Certificate Authority (CA) for `.local` or `.test` domains.
- **Production VPS**: Caddy automatically provisions real, trusted public certificates via ACME (Let's Encrypt & ZeroSSL). Ports `80` and `443` must be reachable by public ACME challenge validation servers.
