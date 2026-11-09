#!/bin/bash
set -euo pipefail

# ==============================================================================
# PRODUCTION DATABASE BACKUP SCRIPT
# Multi-Tenant SaaS Platform
# ==============================================================================

BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
CONTAINER_NAME="${DB_CONTAINER:-tenant_postgres}"
DB_NAME="${POSTGRES_DB:-tenant_domains}"
DB_USER="${POSTGRES_USER:-postgres}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

mkdir -p "${BACKUP_DIR}"

BACKUP_FILE="${BACKUP_DIR}/backup_${DB_NAME}_${TIMESTAMP}.sql.gz"
CHECKSUM_FILE="${BACKUP_FILE}.sha256"

echo "=================================================="
echo "Starting PostgreSQL backup: ${TIMESTAMP}"
echo "Container: ${CONTAINER_NAME}"
echo "Database:  ${DB_NAME}"
echo "Target:    ${BACKUP_FILE}"
echo "=================================================="

# Check container running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "Error: Database container ${CONTAINER_NAME} is not running!" >&2
  exit 1
fi

# Execute pg_dump directly through Docker, piping to gzip
docker exec -t "${CONTAINER_NAME}" pg_dump -U "${DB_USER}" -d "${DB_NAME}" --clean --if-exists --no-owner --no-privileges | gzip -9 > "${BACKUP_FILE}"

# Generate SHA256 checksum for tamper and corruption detection
if command -v sha256sum &> /dev/null; then
  sha256sum "${BACKUP_FILE}" > "${CHECKSUM_FILE}"
elif command -v shasum &> /dev/null; then
  shasum -a 256 "${BACKUP_FILE}" > "${CHECKSUM_FILE}"
fi

BACKUP_SIZE=$(ls -lh "${BACKUP_FILE}" | awk '{print $5}')
echo "Backup completed successfully!"
echo "Size: ${BACKUP_SIZE}"
echo "Checksum file: ${CHECKSUM_FILE}"

# Retention Rotation: Purge backups older than RETENTION_DAYS
echo "Applying retention policy: deleting backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f \( -name "*.sql.gz" -o -name "*.sha256" \) -mtime "+${RETENTION_DAYS}" -exec rm -f {} +
echo "Retention rotation complete."
echo "=================================================="
