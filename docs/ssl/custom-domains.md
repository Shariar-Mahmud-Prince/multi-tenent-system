# Custom Domain Lifecycle & Verification Workflow

## 1. 10-Step Domain Onboarding Lifecycle
1. **Store Owner Input**: The store owner enters their desired custom domain (e.g. `shop.merchant.com` or `merchant.com`) in the merchant dashboard.
2. **Format & Syntax Validation**:
   - Backend normalizes hostname to lowercase.
   - Rejects invalid characters, punycode exploits, IP addresses, and reserved platform domains.
3. **Cryptographic Token Generation**:
   - Backend generates a 32-byte (64 hex characters) cryptographically secure verification token.
   - Token expires in 48 hours.
4. **State Transition**:
   - Domain is recorded in the PostgreSQL database with `domain_status = 'pending_verification'`.
5. **DNS Instructions Generation**:
   - Merchant is provided with:
     - **TXT Verification Record**: Host `_saas-verify.<custom-domain>` with Value `token=<verification_token>`.
     - **Routing Record**: CNAME pointing to `<subdomain>.<base-domain>` (e.g. `mystore.platform.com`), or A record pointing to the VPS IP for apex root domains.
6. **DNS Verification Trigger**:
   - Merchant clicks "Verify Domain" in their dashboard.
7. **Server-Side Verification**:
   - The Go backend's `DNSVerifier` queries authoritative DNS servers for the `_saas-verify` TXT record and CNAME target.
   - If match is confirmed, `domain_status` transitions to `verified`.
8. **TLS Certificate Provisioning**:
   - Caddy's `ask` endpoint now returns HTTP 200 OK for this hostname.
   - On the first incoming TLS handshake, Caddy completes ACME challenge and obtains the certificate.
9. **Storefront Routing Active**:
   - `domain_status` transitions to `active`.
   - Redis cache is primed with tenant domain mapping.
10. **Ongoing Monitoring**:
    - Automatic certificate renewal is managed continuously by Caddy without downtime or manual intervention.

## 2. DNS Configuration Matrix for Customers

| Domain Type | Hostname / Subdomain | Record Type | Target Value | Purpose |
|-------------|----------------------|-------------|--------------|---------|
| Any Custom Domain | `_saas-verify` | **TXT** | `token=<64_CHAR_TOKEN>` | Ownership Verification |
| Subdomain (e.g. `shop.example.com`) | `shop` | **CNAME** | `<store>.yourplatform.com` | Traffic Routing |
| Apex Domain (e.g. `example.com`) | `@` | **A** | `<VPS_PUBLIC_IP>` | Traffic Routing |
