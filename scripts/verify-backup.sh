#!/bin/bash
set -euo pipefail

# ==============================================================================
# AUTOMATED BACKUP INTEGRITY VERIFICATION TEST
# ==============================================================================

BACKUP_DIR="${BACKUP_DIR:-./backups}"
LATEST_BACKUP=$(find "${BACKUP_DIR}" -type f -name "*.sql.gz" | sort | tail -n 1)

if [ -z "${LATEST_BACKUP}" ]; then
  echo "No backup archives found in ${BACKUP_DIR}. Creating a fresh backup for verification..."
  ./scripts/backup.sh
  LATEST_BACKUP=$(find "${BACKUP_DIR}" -type f -name "*.sql.gz" | sort | tail -n 1)
fi

echo "Verifying latest backup archive: ${LATEST_BACKUP}"

# Test gzip integrity
gzip -t "${LATEST_BACKUP}"
echo "Gzip archive test passed: archive is uncorrupted."

# Test SQL header check
HEADER=$(gzip -dc "${LATEST_BACKUP}" | head -n 20)
if echo "${HEADER}" | grep -q "PostgreSQL database dump"; then
  echo "Dump header verified: Valid PostgreSQL dump detected."
else
  echo "Error: Archive does not contain valid PostgreSQL dump header!" >&2
  exit 1
fi

echo "Backup integrity test successfully verified!"
