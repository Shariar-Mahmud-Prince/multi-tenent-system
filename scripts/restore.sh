#!/bin/bash
set -euo pipefail

# ==============================================================================
# PRODUCTION DATABASE RESTORATION SCRIPT
# Multi-Tenant SaaS Platform
# ==============================================================================

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <path_to_backup_file.sql.gz>"
  echo "Example: $0 ./backups/backup_tenant_domains_20261109_200000.sql.gz"
  exit 1
fi

BACKUP_FILE="$1"
CONTAINER_NAME="${DB_CONTAINER:-tenant_postgres}"
DB_NAME="${POSTGRES_DB:-tenant_domains}"
DB_USER="${POSTGRES_USER:-postgres}"

if [ ! -f "${BACKUP_FILE}" ]; then
  echo "Error: Backup file '${BACKUP_FILE}' does not exist!" >&2
  exit 1
fi

CHECKSUM_FILE="${BACKUP_FILE}.sha256"
if [ -f "${CHECKSUM_FILE}" ]; then
  echo "Verifying SHA256 integrity checksum..."
  if command -v sha256sum &> /dev/null; then
    sha256sum -c "${CHECKSUM_FILE}"
  elif command -v shasum &> /dev/null; then
    shasum -a 256 -c "${CHECKSUM_FILE}"
  fi
  echo "Checksum verified successfully!"
else
  echo "Warning: Checksum file '${CHECKSUM_FILE}' not found. Proceeding with unverified archive."
fi

echo "=================================================="
echo "WARNING: DESTRUCTIVE RESTORATION ACTION"
echo "Target Container: ${CONTAINER_NAME}"
echo "Target Database:  ${DB_NAME}"
echo "Source Backup:    ${BACKUP_FILE}"
echo "=================================================="

if [ "${FORCE:-false}" != "true" ]; then
  read -p "Are you sure you want to restore and overwrite the current database? (type 'yes' to proceed): " CONFIRM
  if [ "${CONFIRM}" != "yes" ]; then
    echo "Restoration aborted by user."
    exit 0
  fi
fi

echo "Starting database restoration..."
gzip -dc "${BACKUP_FILE}" | docker exec -i "${CONTAINER_NAME}" psql -U "${DB_USER}" -d "${DB_NAME}"

echo "Restoration complete! Verifying database integrity..."
docker exec -t "${CONTAINER_NAME}" psql -U "${DB_USER}" -d "${DB_NAME}" -c "SELECT COUNT(*) as store_count FROM stores;"

echo "Database restoration finished successfully."
