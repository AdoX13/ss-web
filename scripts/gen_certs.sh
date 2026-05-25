#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECRETS_DIR="${ROOT_DIR}/secrets"
FORCE="${FORCE:-0}"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required to generate certificates" >&2
  exit 1
fi

mkdir -p "${SECRETS_DIR}"
chmod 700 "${SECRETS_DIR}"

required=(ca.key ca.crt server.key server.crt web.key web.crt device.key device.crt)
if [[ "${FORCE}" != "1" ]]; then
  for file in "${required[@]}"; do
    if [[ -e "${SECRETS_DIR}/${file}" ]]; then
      echo "${SECRETS_DIR}/${file} already exists. Set FORCE=1 to rotate local demo certs." >&2
      exit 1
    fi
  done
fi

server_ext="$(mktemp)"
client_ext="$(mktemp)"
trap 'rm -f "${server_ext}" "${client_ext}" "${SECRETS_DIR}"/*.csr' EXIT

cat >"${server_ext}" <<'EOF'
subjectAltName=DNS:broker,DNS:localhost,IP:127.0.0.1
extendedKeyUsage=serverAuth
keyUsage=digitalSignature,keyEncipherment
EOF

cat >"${client_ext}" <<'EOF'
extendedKeyUsage=clientAuth
keyUsage=digitalSignature
EOF

openssl req -x509 -newkey rsa:4096 -nodes -sha256 -days 3650 \
  -keyout "${SECRETS_DIR}/ca.key" \
  -out "${SECRETS_DIR}/ca.crt" \
  -subj "/CN=MedSecOCR Local CA"

openssl req -newkey rsa:2048 -nodes \
  -keyout "${SECRETS_DIR}/server.key" \
  -out "${SECRETS_DIR}/server.csr" \
  -subj "/CN=broker"
openssl x509 -req -sha256 -days 825 \
  -in "${SECRETS_DIR}/server.csr" \
  -CA "${SECRETS_DIR}/ca.crt" \
  -CAkey "${SECRETS_DIR}/ca.key" \
  -CAcreateserial \
  -out "${SECRETS_DIR}/server.crt" \
  -extfile "${server_ext}"

for name in web device; do
  openssl req -newkey rsa:2048 -nodes \
    -keyout "${SECRETS_DIR}/${name}.key" \
    -out "${SECRETS_DIR}/${name}.csr" \
    -subj "/CN=${name}"
  openssl x509 -req -sha256 -days 825 \
    -in "${SECRETS_DIR}/${name}.csr" \
    -CA "${SECRETS_DIR}/ca.crt" \
    -CAkey "${SECRETS_DIR}/ca.key" \
    -CAcreateserial \
    -out "${SECRETS_DIR}/${name}.crt" \
    -extfile "${client_ext}"
done

chmod 600 "${SECRETS_DIR}"/*.key
chmod 644 "${SECRETS_DIR}"/*.crt

echo "Generated local mTLS material in ${SECRETS_DIR}."
echo "Use FORCE=1 ${0} to rotate local demo certificates."
