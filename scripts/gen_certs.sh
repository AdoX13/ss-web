#!/usr/bin/env bash
# ==============================================================================
# gen_certs.sh — Generate TLS certificates for MedSec-OCR mTLS (Lab 7)
#
# Creates a self-signed CA and issues server + client certificates.
# Run once during initial setup, then use rotate_certs.sh for renewals.
#
# Usage:
#   ./scripts/gen_certs.sh              # Generate all certs
#   ./scripts/gen_certs.sh --force      # Overwrite existing certs
#
# Output directory: secrets/
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SECRETS_DIR="${PROJECT_ROOT}/secrets"
BACKUP_DIR="${SECRETS_DIR}/backup-$(date +%Y%m%d-%H%M%S)"

# Certificate parameters (Lab 7 requirements)
CA_KEY_BITS=4096
CERT_KEY_BITS=2048
CA_DAYS=3650      # 10 years
CERT_DAYS=365     # 1 year

# Subject components
COUNTRY="RO"
STATE="Romania"
LOCALITY="Bucharest"
ORG="MedSec-OCR"

FORCE=false
if [[ "${1:-}" == "--force" ]]; then
    FORCE=true
fi

# Check if certs already exist
if [[ -f "${SECRETS_DIR}/ca.crt" && "${FORCE}" == "false" ]]; then
    echo "⚠  Certificates already exist in ${SECRETS_DIR}/"
    echo "   Use --force to regenerate (existing certs will be backed up)"
    echo ""
    echo "   Current CA cert info:"
    openssl x509 -in "${SECRETS_DIR}/ca.crt" -noout -subject -enddate 2>/dev/null || echo "   (could not read CA cert)"
    exit 1
fi

# Backup existing certs if they exist
if [[ -f "${SECRETS_DIR}/ca.crt" ]]; then
    echo "📦 Backing up existing certificates to ${BACKUP_DIR}/"
    mkdir -p "${BACKUP_DIR}"
    cp -p "${SECRETS_DIR}"/*.crt "${SECRETS_DIR}"/*.key "${BACKUP_DIR}/" 2>/dev/null || true
    cp -p "${SECRETS_DIR}"/*.srl "${BACKUP_DIR}/" 2>/dev/null || true
fi

mkdir -p "${SECRETS_DIR}"

echo "🔐 Generating MedSec-OCR TLS certificates (Lab 7)"
echo "   Output: ${SECRETS_DIR}/"
echo ""

# ---- CA (Certificate Authority) ----
echo "1/5  Generating CA key (${CA_KEY_BITS}-bit RSA)..."
openssl genrsa -out "${SECRETS_DIR}/ca.key" ${CA_KEY_BITS} 2>/dev/null

echo "     Generating CA certificate (valid ${CA_DAYS} days)..."
openssl req -new -x509 -days ${CA_DAYS} \
    -key "${SECRETS_DIR}/ca.key" \
    -out "${SECRETS_DIR}/ca.crt" \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG}/OU=Security/CN=MedSec-CA"

# ---- Server cert (MQTT broker) ----
echo "2/5  Generating broker server key (${CERT_KEY_BITS}-bit RSA)..."
openssl genrsa -out "${SECRETS_DIR}/server.key" ${CERT_KEY_BITS} 2>/dev/null

echo "     Generating broker CSR with SANs..."
openssl req -new \
    -key "${SECRETS_DIR}/server.key" \
    -out "${SECRETS_DIR}/server.csr" \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG}/OU=Broker/CN=broker" \
    -addext "subjectAltName = DNS:broker,DNS:localhost,IP:127.0.0.1"

echo "     Signing broker certificate with CA..."
openssl x509 -req -days ${CERT_DAYS} \
    -in "${SECRETS_DIR}/server.csr" \
    -CA "${SECRETS_DIR}/ca.crt" \
    -CAkey "${SECRETS_DIR}/ca.key" \
    -CAcreateserial \
    -copy_extensions copyall \
    -out "${SECRETS_DIR}/server.crt" 2>/dev/null

# ---- Web client cert (Go API) ----
echo "3/5  Generating web client key (Go API)..."
openssl genrsa -out "${SECRETS_DIR}/web.key" ${CERT_KEY_BITS} 2>/dev/null

echo "     Generating web client CSR..."
openssl req -new \
    -key "${SECRETS_DIR}/web.key" \
    -out "${SECRETS_DIR}/web.csr" \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG}/OU=WebClient/CN=web"

echo "     Signing web client certificate with CA..."
openssl x509 -req -days ${CERT_DAYS} \
    -in "${SECRETS_DIR}/web.csr" \
    -CA "${SECRETS_DIR}/ca.crt" \
    -CAkey "${SECRETS_DIR}/ca.key" \
    -CAcreateserial \
    -out "${SECRETS_DIR}/web.crt" 2>/dev/null

# ---- Python sender cert (ingestion scripts) ----
echo "4/5  Generating Python sender client key..."
openssl genrsa -out "${SECRETS_DIR}/python-sender.key" ${CERT_KEY_BITS} 2>/dev/null

echo "     Generating Python sender CSR..."
openssl req -new \
    -key "${SECRETS_DIR}/python-sender.key" \
    -out "${SECRETS_DIR}/python-sender.csr" \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG}/OU=Ingestion/CN=python-sender"

echo "     Signing Python sender certificate with CA..."
openssl x509 -req -days ${CERT_DAYS} \
    -in "${SECRETS_DIR}/python-sender.csr" \
    -CA "${SECRETS_DIR}/ca.crt" \
    -CAkey "${SECRETS_DIR}/ca.key" \
    -CAcreateserial \
    -out "${SECRETS_DIR}/python-sender.crt" 2>/dev/null

# ---- Cleanup ----
echo "5/5  Cleaning up CSR files and setting permissions..."
rm -f "${SECRETS_DIR}"/*.csr

# Secure file permissions
chmod 600 "${SECRETS_DIR}"/*.key
chmod 644 "${SECRETS_DIR}"/*.crt
chmod 644 "${SECRETS_DIR}"/*.srl 2>/dev/null || true

echo ""
echo "✅ Certificate generation complete!"
echo ""
echo "   Certificate chain:"
echo "   ┌─ CA (MedSec-CA)"
echo "   ├── broker  (server.crt)  — MQTT broker identity"
echo "   ├── web     (web.crt)     — Go API MQTT client"
echo "   └── python-sender         — Python ingestion scripts"
echo ""
echo "   Certificate details:"
echo "   ─────────────────────────────────────────────────────"
printf "   %-25s %-20s %s\n" "File" "Subject CN" "Expires"
echo "   ─────────────────────────────────────────────────────"

for cert in ca server web python-sender; do
    if [[ -f "${SECRETS_DIR}/${cert}.crt" ]]; then
        cn=$(openssl x509 -in "${SECRETS_DIR}/${cert}.crt" -noout -subject 2>/dev/null | sed 's/.*CN *= *//')
        expiry=$(openssl x509 -in "${SECRETS_DIR}/${cert}.crt" -noout -enddate 2>/dev/null | sed 's/notAfter=//')
        printf "   %-25s %-20s %s\n" "${cert}.crt" "${cn}" "${expiry}"
    fi
done

echo ""
echo "   Next steps:"
echo "   1. Run: docker compose down && docker compose up -d"
echo "   2. Test: mosquitto_sub -h localhost -p 8883 --cafile secrets/ca.crt \\"
echo "            --cert secrets/web.crt --key secrets/web.key -t '\$SYS/#' -C 1"
echo "   3. Test: python3 scripts/send_image.py"
