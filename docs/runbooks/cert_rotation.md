# Certificate Rotation Runbook

## Overview
This runbook describes the procedure for rotating the TLS certificates used for mutual TLS (mTLS) in the MedSec-OCR project. The CA certificate has a 10-year validity, while server and client certificates are valid for 1 year.

## When to Rotate
- **Routine**: Expiry approaching within 30 days.
- **Compromise**: Key compromised or suspected of being compromised.
- **Personnel Change**: Departure of a team member with access to the client or server keys.

## Prerequisites
- Local environment with `openssl` installed.
- Access to the CA key (`secrets/ca.key`). **Ensure this key is kept secure.**

## Procedure

### 1. Routine Rotation (Using `rotate_certs.sh`)
The recommended way to rotate certificates is using the automated script:

```bash
./scripts/rotate_certs.sh --verify
```
This script will:
1. Backup existing certs to `secrets/backup-YYYYMMDD-HHMMSS/`.
2. Generate new server and client certificates signed by the existing CA.
3. Restart the Docker Compose services (`broker`, `go-api`).
4. Verify the new certificates with a test MQTT connection.

### 2. Manual Rotation
If you need to rotate a specific certificate manually (e.g., just the web client):

1. Back up the existing key/cert.
2. Generate a new key and CSR:
   ```bash
   openssl genrsa -out secrets/web.key 2048
   openssl req -new -key secrets/web.key -out secrets/web.csr -subj "/C=RO/ST=Romania/L=Bucharest/O=MedSec-OCR/OU=WebClient/CN=web"
   ```
3. Sign the CSR with the CA:
   ```bash
   openssl x509 -req -days 365 -in secrets/web.csr -CA secrets/ca.crt -CAkey secrets/ca.key -CAcreateserial -out secrets/web.crt
   ```
4. Restart the dependent services (e.g., `go-api`).

## Verification
You can manually test the new certificates by connecting to the broker:

```bash
mosquitto_sub -h localhost -p 8883 --cafile secrets/ca.crt --cert secrets/web.crt --key secrets/web.key -t '$SYS/#' -C 1
```

## Rollback Procedure
If the new certificates fail validation or services refuse to start:
1. Restore the backup directory:
   ```bash
   cp -p secrets/backup-YYYYMMDD-HHMMSS/* secrets/
   ```
2. Restart the services:
   ```bash
   docker compose restart broker go-api
   ```

## Emergency CA Rotation
If the CA key is compromised, **ALL** certificates must be regenerated:
1. Revoke access and shut down the broker.
2. Run `./scripts/gen_certs.sh --force`.
3. Distribute the new CA to all clients.
4. Restart all services.
