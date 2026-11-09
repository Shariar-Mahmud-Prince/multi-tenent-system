# System Architecture & Single-VPS Infrastructure Specification

**Architecture Version**: 2.0  
**Target Deployment**: Single Production VPS (e.g., 4 vCPU, 8GB RAM, 80GB NVMe SSD)  
**Future Scaling**: Zero-rewrite migration path to multi-node cluster

---

## 1. Physical & Network Topology

```
Public Internet
       │
       ├───────────────────────────────────────┐
       ▼ (:80 HTTP)                            ▼ (:443 HTTPS)
┌─────────────────────────────────────────────────────────────┐
│                       CADDY PROXY                           │
│  - Public Ingress Points: :80, :443/TCP, :443/UDP (HTTP/3)  │
│  - Automatic On-Demand TLS Issuance via Let's Encrypt / CA  │
│  - Automatic HTTP -> HTTPS Canonical Redirection            │
│  - Strict Reverse Proxy Headers                             │
└──────────────────────────────┬──────────────────────────────┘
                               │
               Internal Bridge Network (tenant_net)
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│    GO API    │       │    REDIS 7   │       │  POSTGRES 16 │
│  (Port 8080) │◄─────►│ (Port 6379)  │       │ (Port 5432)  │
│  - Chi Router│       │- Token Bucket│       │- Tenants &   │
│  - JWT Auth  │       │- Cache Tier  │       │  Domains     │
│  - DNS Check │       │- Tenant Keys │       │- B-Tree Idx  │
└──────────────┘       └──────────────┘       └──────┬───────┘
       ▲                                             │
       │                                             │
       ├─────────────────────────────────────────────┘
       │
┌──────┴──────────────────────────────────────────────────────┐
│              MONITORING PROFILE (Protected)                 │
│  - Prometheus (:9090 - internal only)                       │
│  - Grafana (:3000 - authenticated via admin credentials)    │
│  - Node Exporter (:9100 - host metrics, internal only)      │
│  - cAdvisor (:8081 - container metrics, internal only)      │
│  - Postgres Exporter (:9187 - DB metrics, internal only)    │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Port Allocation & Firewall Rules

| Service | Port | Scope | Access Policy |
| :--- | :--- | :--- | :--- |
| **Caddy HTTP** | `80/tcp` | **Public** | Public internet for ACME HTTP-01 challenge and HTTPS redirects. |
| **Caddy HTTPS** | `443/tcp`, `443/udp` | **Public** | Public internet for secure storefront and API traffic (HTTP/1.1, HTTP/2, HTTP/3 QUIC). |
| **Go Backend API** | `8080/tcp` | **Private** | Internal Docker network (`tenant_net`) only. Accessible exclusively via Caddy proxy. |
| **Redis Cache** | `6379/tcp` | **Private** | Internal Docker network (`tenant_net`) only. Password protected (`requirepass`). |
| **PostgreSQL DB** | `5432/tcp` | **Private** | Internal Docker network (`tenant_net`) only. Not exposed to host. |
| **Grafana UI** | `3000/tcp` | **Protected** | Bound to localhost (`127.0.0.1:3000`) or reverse proxied through Caddy at `monitor.yourplatform.com` behind authentication. |
| **Prometheus** | `9090/tcp` | **Private** | Internal Docker network only. |
| **Metrics Exporters** | `9100`, `8081`, `9187` | **Private** | Internal Docker network only. |

---

## 3. VPS Resource Budget & Allocation

Designed for an **8GB RAM / 4 vCPU** single production host:

| Service | Memory Limit (`mem_limit`) | Memory Reservation | CPU Quota (`cpus`) | Restart Policy |
| :--- | :--- | :--- | :--- | :--- |
| **Caddy** | `512 MB` | `128 MB` | `1.0` | `unless-stopped` |
| **Go API (`app`)** | `1024 MB` | `256 MB` | `1.5` | `unless-stopped` |
| **PostgreSQL 16** | `2048 MB` | `512 MB` | `1.5` | `unless-stopped` |
| **Redis 7** | `512 MB` | `128 MB` | `0.5` | `unless-stopped` |
| **Prometheus** | `1024 MB` | `256 MB` | `0.5` | `unless-stopped` |
| **Grafana** | `512 MB` | `128 MB` | `0.5` | `unless-stopped` |
| **Exporters (Combined)** | `512 MB` | `128 MB` | `0.5` | `unless-stopped` |
| **Host OS & Buffer** | **~2000 MB** | — | — | — |
| **Total Allocation** | **6144 MB max** | **1536 MB min** | — | — |

> [!IMPORTANT]
> The total container memory limit caps at ~6.1 GB, ensuring **~2 GB of free physical RAM** remains available for the Linux operating system kernel, page cache, disk I/O buffers, and emergency spikes.

---

## 4. Docker Compose Profiles

The infrastructure uses Docker Compose profiles to prevent testing or monitoring tools from consuming resources during lightweight operation:

1. **`core`** (Default): `caddy`, `app`, `postgres`, `redis`.
2. **`monitoring`**: `prometheus`, `grafana`, `node_exporter`, `cadvisor`, `postgres_exporter`.
   - Command: `docker compose --profile monitoring up -d`
3. **`testing`**: `k6` load testing runner.
   - Command: `docker compose --profile testing run --rm k6 run /scripts/load-test.js`
