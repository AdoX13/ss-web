# Disaster Recovery Runbook

## Overview
This runbook outlines the steps to recover the MedSec-OCR environment from catastrophic failure, including data loss, server failure, or widespread compromise.

## Objectives
- **Recovery Time Objective (RTO)**: 4 hours
- **Recovery Point Objective (RPO)**: 24 hours (based on daily backups)

## Backup Procedures

### MongoDB Backups
Backups must be taken daily using `mongodump`. Since MongoDB runs in Docker, execute:

```bash
# Create a backup archive
docker compose exec mongo-db mongodump \
  --username=admin \
  --password=supersecret \
  --authenticationDatabase=admin \
  --archive=/data/db/backup.archive

# Copy it to the host
docker cp mongo-db:/data/db/backup.archive ./backups/backup-$(date +%F).archive
```

### Certificate Backups
Ensure that `secrets/ca.key` and `secrets/ca.crt` are securely backed up offline. Without the CA key, all clients will need to be re-provisioned with new certificates if the server goes down.

## Recovery Procedures

### 1. Full Stack Rebuild
If the host server is lost, rebuild the environment from scratch:

1. Clone the repository to the new host.
2. Restore `.env` from your secure password manager.
3. Restore the `secrets/` directory (specifically the CA files).
4. Run certificate generation (if server certs were lost):
   ```bash
   ./scripts/gen_certs.sh
   ```
5. Start the environment:
   ```bash
   ./start.sh
   ```

### 2. Database Restoration
To restore MongoDB from an archive:

1. Ensure the database container is running.
2. Copy the backup to the container:
   ```bash
   docker cp ./backups/backup-latest.archive mongo-db:/data/db/restore.archive
   ```
3. Run `mongorestore`:
   ```bash
   docker compose exec mongo-db mongorestore \
     --username=admin \
     --password=supersecret \
     --authenticationDatabase=admin \
     --archive=/data/db/restore.archive \
     --drop
   ```
   *Note: The `--drop` flag will delete existing collections before restoring.*

## Disaster Recovery Drills
DR drills should be conducted quarterly:
1. Spin up a secondary environment.
2. Restore the latest backup.
3. Verify application functionality and data integrity.
4. Document the time taken and any issues encountered.
