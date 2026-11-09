# Database Backup & Disaster Recovery Architecture

## 1. Objectives & Metrics
- **Recovery Point Objective (RPO)**: < 1 hour (Maximum data loss allowable in a catastrophic event).
- **Recovery Time Objective (RTO)**: < 15 minutes (Maximum time required to fully restore database service and resume customer transactions).

## 2. Backup Strategy
- **Mechanism**: Compressed, encrypted logical dumps via `pg_dump` with `--clean --if-exists --no-owner --no-privileges`.
- **Compression**: `gzip -9` stream compression.
- **Integrity**: SHA256 checksum generated concurrently with every dump (`.sql.gz.sha256`).
- **Storage**:
  - Primary Local: Dedicated host directory `./backups/` mounted on persistent host disk.
  - Secondary Off-Site: Periodic sync to remote S3-compatible cloud storage (e.g. AWS S3 / Cloudflare R2 / Backblaze B2) using `rclone` or AWS CLI.
- **Retention Schedule**:
  - Hourly snapshots: Retained for 24 hours.
  - Daily backups: Retained for 30 days.
  - Monthly archives: Retained for 1 year.

## 3. Automation via Cron
Install the following root cron job on the production VPS:
```bash
# Execute automated backup every hour at minute 0
0 * * * * cd /opt/saas-ecommerce && ./scripts/backup.sh >> /var/log/saas-backup.log 2>&1

# Verify backup integrity daily at 03:00 UTC
0 3 * * * cd /opt/saas-ecommerce && ./scripts/verify-backup.sh >> /var/log/saas-backup-verify.log 2>&1
```

## 4. Disaster Recovery Procedure
In the event of database corruption or hardware failure:
1. Identify the latest valid backup archive:
   ```bash
   ls -lt ./backups/*.sql.gz | head -n 1
   ```
2. Validate SHA256 checksum:
   ```bash
   sha256sum -c ./backups/backup_tenant_domains_YYYYMMDD_HHMMSS.sql.gz.sha256
   ```
3. Execute the restoration script:
   ```bash
   ./scripts/restore.sh ./backups/backup_tenant_domains_YYYYMMDD_HHMMSS.sql.gz
   ```
4. Verify application health:
   ```bash
   curl -s http://127.0.0.1:8080/healthz
   ```
