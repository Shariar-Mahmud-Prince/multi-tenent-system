# Certificate Troubleshooting & Diagnostic Runbook

## 1. Common Issues & Resolutions

### Issue 1: Browser Shows "Not Secure" / Self-Signed Warning on `.local` Domains
- **Root Cause**: `.local` is not a public top-level domain. Let's Encrypt cannot issue public certificates for private domains. In local development, Caddy generates an internal certificate via its internal CA (`issuer internal`).
- **Resolution**:
  - For local Windows testing, import Caddy's root CA certificate into the Windows Trusted Root Certification Authorities store:
    ```cmd
    docker cp tenant_caddy:/data/caddy/pki/authorities/local/root.crt caddy_root.crt
    certutil -addstore -f "ROOT" caddy_root.crt
    ```
  - In production VPS with real domains (e.g. `.com`, `.store`), Caddy automatically requests official public certificates from Let's Encrypt / ZeroSSL.

### Issue 2: Caddy Log: `on-demand TLS rate limit exceeded` or `ask endpoint returned 403`
- **Root Cause**: The domain has not yet completed cryptographic DNS ownership verification, or the store was deactivated.
- **Diagnostic Step**:
  ```bash
  curl -i "http://127.0.0.1:8080/api/v1/domains/validate?domain=<custom-domain>"
  ```
- **Resolution**:
  - Verify that `domain_status` is `verified` or `active` in the database:
    ```sql
    SELECT name, custom_domain, domain_status FROM stores WHERE custom_domain = '<custom-domain>';
    ```
  - Ask the merchant to click "Verify Domain" after verifying their DNS TXT record.

### Issue 3: Let's Encrypt ACME Challenge Fails
- **Root Cause**: Firewall blocking inbound port 80/443, or DNS records have not fully propagated.
- **Diagnostic Step**:
  ```bash
  docker logs tenant_caddy --tail 100
  ```
- **Resolution**:
  - Verify public VPS security group allows inbound TCP ports `80` and `443`.
  - Check DNS propagation with `dig +trace <domain>`.
  - Ensure customer CAA records (if present) permit `letsencrypt.org` and `zerossl.com`.
