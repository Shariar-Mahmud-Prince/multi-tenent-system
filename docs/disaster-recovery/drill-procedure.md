# Disaster Recovery Drill Procedure (Quarterly Verification Runbook)

## Purpose
To validate that production databases and multi-tenant stores can be reconstructed from zero within the 15-minute RTO without data loss beyond the 1-hour RPO.

## Schedule
Executed quarterly by DevOps/SRE engineers on a dedicated staging or sandbox VPS.

## Step-by-Step Drill Execution

### Phase 1: Preparation & Snapshot
1. Trigger an on-demand snapshot:
   ```bash
   ./scripts/backup.sh
   ```
2. Note the generated archive filename and sha256 checksum.

### Phase 2: Simulating Failure
1. Simulate total database loss:
   ```bash
   docker exec -t tenant_postgres psql -U postgres -c "DROP DATABASE tenant_domains;"
   docker exec -t tenant_postgres psql -U postgres -c "CREATE DATABASE tenant_domains;"
   ```
2. Verify API failure:
   ```bash
   curl -I http://127.0.0.1:8080/api/v1/stores
   # Should return 500 Internal Server Error
   ```

### Phase 3: Restoration
1. Run restoration script with FORCE mode in drill environment:
   ```bash
   FORCE=true ./scripts/restore.sh ./backups/<LATEST_BACKUP>.sql.gz
   ```
2. Inspect log output for successful table recreation and row counts.

### Phase 4: Post-Restoration Acceptance Verification
1. Run automated test suites:
   ```bash
   go test -v ./tests/e2e/store/...
   go test -v ./tests/security/...
   ```
2. Query health endpoint:
   ```bash
   curl -s http://127.0.0.1:8080/healthz
   ```
3. Record duration from disaster start to system recovery:
   - Target: < 15 minutes.
   - Record in SRE Incident & Drill Logbook.
