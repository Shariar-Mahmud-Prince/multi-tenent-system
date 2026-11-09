# Incident Response & On-Call Triage Runbook

## 1. Severity Levels
- **SEV-1 (Critical Outage)**: System storefronts or API completely unreachable for multiple tenants. Response SLA: < 15 minutes.
- **SEV-2 (Degraded Performance)**: High latency (>1s) or elevated error rate (>5%). Response SLA: < 30 minutes.
- **SEV-3 (Minor / Single Tenant)**: Isolated domain verification failure or non-critical background issue. Response SLA: < 4 hours.

## 2. Common Scenarios & Immediate Actions

### Scenario A: API Unreachable (HTTP 502 Bad Gateway via Caddy)
1. Verify container status:
   ```bash
   docker ps -a
   ```
2. Check application logs for crash or panic:
   ```bash
   docker logs tenant_app --tail 200
   ```
3. Restart application container if necessary:
   ```bash
   docker compose restart app
   ```

### Scenario B: Database Connection Pool Exhausted
1. Inspect active Postgres connections:
   ```bash
   docker exec -it tenant_postgres psql -U postgres -d tenant_domains -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"
   ```
2. Terminate long-running or idle-in-transaction connections:
   ```bash
   docker exec -it tenant_postgres psql -U postgres -d tenant_domains -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle in transaction' AND state_change < current_timestamp - interval '5 minutes';"
   ```

### Scenario C: Malicious Bot Traffic or DoS Attack
1. Identify high-frequency IP addresses:
   ```bash
   docker logs tenant_caddy | awk '{print $1}' | sort | uniq -c | sort -nr | head -n 20
   ```
2. Temporary IP block via iptables or UFW:
   ```bash
   sudo ufw deny from <OFFENDING_IP> to any
   ```
3. Verify that `RateLimitMiddleware` (50 req/s, burst 100) and `BotControlMiddleware` are returning HTTP 429 / 403.
