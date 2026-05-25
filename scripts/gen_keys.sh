#!/usr/bin/env bash

set -euo pipefail

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required to generate development keys" >&2
  exit 1
fi

cat <<EOF
# Add these values to .env or your deployment secret store.
JWT_SECRET=$(openssl rand -base64 32)
MEDSEC_MASTER_KEY=$(openssl rand -base64 32)
EVIDENCE_ED25519_PRIVATE_KEY=$(openssl rand -base64 32)
EOF
