#!/usr/bin/env bash
# ==============================================================================
# rotate_certs.sh — Rotate client/server TLS certificates (keep CA stable)
#
# Regenerates server and client certificates signed by the existing CA.
# The CA key and certificate are NOT rotated (they have a 10-year lifetime).
#
# Usage:
#   ./scripts/rotate_certs.sh                  # Rotate all client/server certs
#   ./scripts/rotate_certs.sh --restart        # Also restart Docker services
#   ./scripts/rotate_certs.sh --verify         # Rotate, restart, and verify
#
# See also: docs/runbooks/cert_rotation.md
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SECRETS_DIR="${PROJECT_ROOT}/secrets"
BACKUP_DIR="${SECRETS_DIR}/backup-$(date +%Y%m%d-%H%M%S)"

CERT_KEY_BITS=2048
CERT_DAYS=365

COUNTRY="RO"
STATE="Romania"
LOCALITY="Bucharest"
ORG="MedSec-OCR"

DO_RESTART=false
DO_VERIFY=false

for arg in "$@"; do
    case "$arg" in
        --restart) DO_RESTART=true ;;
        --verify)  DO_RESTART=true; DO_VERIFY=true ;;
        *)         echo "Unknown argument: $arg"; exit 1 ;;
    esac
done

# Pre-flight checks
if [[ ! -f "${SECRETS_DIR}/ca.key" ]]; then
    echo "✗ CA key not found at ${SECRETS_DIR}/ca.key"
    echo "  Run gen_certs.sh first to create the CA."
    exit 1
fi

if [[ ! -f "${SECRETS_DIR}/ca.crt" ]]; then
    echo "✗ CA certificate not found at ${SECRETS_DIR}/ca.crt"
    exit 1
fi

echo "🔄 Rotating TLS certificates (keeping CA stable)"
echo ""

# Show current cert expiry dates
echo "   Current certificate status:"
for cert in server web python-sender; do
    if [[ -f "${SECRETS_DIR}/${cert}.crt" ]]; then
        expiry=$(openssl x509 -in "${SECRETS_DIR}/${cert}.crt" -noout -enddate 2>/dev/null | sed 's/notAfter=//')
        printf "   %-25s expires: %s\n" "${cert}.crt" "${expiry}"
    fi
done
echo ""

# Backup current certs
echo "📦 Backing up current certificates to ${BACKUP_DIR}/"
mkdir -p "${BACKUP_DIR}"
cp -p "${SECRETS_DIR}"/*.crt "${BACKUP_DIR}/" 2>/dev/null || true
cp -p "${SECRETS_DIR}"/*.key "${BACKUP_DIR}/" 2>/dev/null || true
cp -p "${SECRETS_DIR}"/*.srl "${BACKUP_DIR}/" 2>/dev/null || true

# Function to generate a cert signed by the CA
generate_cert() {
    local name="$1"
    local ou="$2"
    local cn="$3"

    echo "   Generating ${name} key + cert..."
    openssl genrsa -out "${SECRETS_DIR}/${name}.key" ${CERT_KEY_BITS} 2>/dev/null
    openssl req -new \
        -key "${SECRETS_DIR}/${name}.key" \
        -out "${SECRETS_DIR}/${name}.csr" \
        -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG}/OU=${ou}/CN=${cn}" 2>/dev/null
    openssl x509 -req -days ${CERT_DAYS} \
        -in "${SECRETS_DIR}/${name}.csr" \
        -CA "${SECRETS_DIR}/ca.crt" \
        -CAkey "${SECRETS_DIR}/ca.key" \
        -CAcreateserial \
        -out "${SECRETS_DIR}/${name}.crt" 2>/dev/null
    rm -f "${SECRETS_DIR}/${name}.csr"
    chmod 600 "${SECRETS_DIR}/${name}.key"
    chmod 644 "${SECRETS_DIR}/${name}.crt"
}

echo "🔐 Regenerating certificates..."
generate_cert "server"        "Broker"     "broker"
generate_cert "web"           "WebClient"  "web"
generate_cert "python-sender" "Ingestion"  "python-sender"

echo ""
echo "   New certificate expiry dates:"
for cert in server web python-sender; do
    expiry=$(openssl x509 -in "${SECRETS_DIR}/${cert}.crt" -noout -enddate 2>/dev/null | sed 's/notAfter=//')
    printf "   %-25s expires: %s\n" "${cert}.crt" "${expiry}"
done

# Verify certificates are signed by our CA
echo ""
echo "🔍 Verifying certificate chain..."
all_valid=true
for cert in server web python-sender; do
    if openssl verify -CAfile "${SECRETS_DIR}/ca.crt" "${SECRETS_DIR}/${cert}.crt" >/dev/null 2>&1; then
        echo "   ✅ ${cert}.crt — valid, signed by CA"
    else
        echo "   ❌ ${cert}.crt — INVALID"
        all_valid=false
    fi
done

if [[ "${all_valid}" == "false" ]]; then
    echo ""
    echo "❌ Some certificates failed validation!"
    echo "   Restoring from backup: ${BACKUP_DIR}/"
    cp -p "${BACKUP_DIR}"/* "${SECRETS_DIR}/" 2>/dev/null || true
    echo "   Rollback complete. Investigate the issue."
    exit 1
fi

# Restart services if requested
if [[ "${DO_RESTART}" == "true" ]]; then
    echo ""
    echo "🐳 Restarting Docker services to pick up new certs..."
    cd "${PROJECT_ROOT}"
    docker compose restart broker go-api
    echo "   Waiting 10 seconds for services to stabilize..."
    sleep 10
fi

# Verify connectivity if requested
if [[ "${DO_VERIFY}" == "true" ]]; then
    echo ""
    echo "🧪 Verifying MQTT connectivity with new certs..."
    if timeout 10 mosquitto_sub -h localhost -p 8883 \
        --cafile "${SECRETS_DIR}/ca.crt" \
        --cert "${SECRETS_DIR}/web.crt" \
        --key "${SECRETS_DIR}/web.key" \
        -t '$SYS/broker/uptime' -C 1 -W 5 >/dev/null 2>&1; then
        echo "   ✅ MQTT connection successful with new certificates"
    else
        echo "   ❌ MQTT connection FAILED with new certificates"
        echo "   Rolling back..."
        cp -p "${BACKUP_DIR}"/* "${SECRETS_DIR}/" 2>/dev/null || true
        docker compose restart broker go-api
        echo "   Rollback complete. Check broker logs: docker compose logs broker"
        exit 1
    fi
fi

echo ""
echo "✅ Certificate rotation complete!"
echo "   Backup stored in: ${BACKUP_DIR}/"
if [[ "${DO_RESTART}" == "false" ]]; then
    echo ""
    echo "   ⚠  Services were NOT restarted."
    echo "   Run: docker compose restart broker go-api"
    echo "   Or:  ./scripts/rotate_certs.sh --verify"
fi
