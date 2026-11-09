# Monitoring, Observability & Alerting Architecture

## 1. Metrics Stack Overview
The observability architecture is built around Prometheus and Grafana running in an isolated Docker network without public exposure:

- **Prometheus**: Scrapes Go API (`/metrics`), Postgres Exporter (`:9187`), Node Exporter (`:9100`), and cAdvisor (`:8080`) at 15-second intervals.
- **Grafana**: Visualizes request throughput, p50/p95/p99 latencies, database connections, and container memory.
- **Privacy Enforcement**: Monitoring ports (`9090`, `3000`, `9187`, `9100`) bind strictly to `127.0.0.1`. Remote administration is accessed via SSH tunneling (`ssh -L 3000:localhost:3000 user@vps_ip`).

## 2. Key Metrics Exposed by Go Application

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `http_requests_total` | Counter | `method`, `path`, `status` | Total processed HTTP requests across the SaaS platform |
| `http_request_duration_seconds` | Histogram | `method`, `path` | Request latency distribution (buckets: 1ms to 10s) |
| `saas_active_tenants` | Gauge | None | Number of active tenant stores cached in memory/Redis |

## 3. Configured Alert Rules (`prometheus/alert_rules.yml`)

1. **HighHttp5xxErrorRate**:
   - Triggers when >5% of HTTP responses are 5xx errors over a 2-minute window.
   - Severity: `critical`.
2. **SlowApiResponses**:
   - Triggers when p95 request duration exceeds 1.0 second over 3 minutes.
   - Severity: `warning`.
3. **PostgresServiceDown**:
   - Triggers immediately when `pg_up == 0`.
   - Severity: `disaster`.
4. **RedisMemoryHigh**:
   - Triggers when Redis memory usage exceeds 85% of allocated memory.
   - Severity: `warning`.
5. **HighContainerCpuUsage**:
   - Triggers when sustained container CPU usage exceeds 85% for 5 minutes.
   - Severity: `warning`.
