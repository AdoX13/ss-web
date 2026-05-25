# TARA Asset Inventory

Scope: MedSec-OCR web platform from Lab 1, secured with Lab 7 mTLS and extended with Lab 10 security reporting.

| Asset | Location | Security goal | P6 control |
|---|---|---|---|
| PHI in MongoDB | `photos`, `patients`, `medical_records` | Confidentiality, integrity | AES-256-GCM field envelopes, HMAC CNP lookup, RBAC |
| Raw medical images | `uploads/photos`, legacy `photos` metadata | Confidentiality, integrity | RBAC, audit events, future encrypted object storage |
| JWT secret | `.env`, runtime env | Authentication integrity | Environment-only secret, no hardcoded production fallback |
| Master encryption key | `MEDSEC_MASTER_KEY` | PHI confidentiality | 32-byte key, never committed, rotation runbook |
| Evidence signing key | `EVIDENCE_ED25519_PRIVATE_KEY` | Non-repudiation | Ed25519 signatures over hash-chain records |
| mTLS CA private key | `secrets/ca.key` local only | MQTT trust root | Not committed, short-lived leaf certs, rotation |
| Audit trail | `audit_log` | Accountability | Append-only writer API, indexed query endpoint |
| Evidence chain | `evidence_chain` | Tamper evidence | Sequential hash chain and Ed25519 verification |
| OCR worker socket | `/run/ocr/ocr.sock` | Service boundary integrity | Unix socket, gVisor/no-network target, size limits |
| Report exports | `/api/v1/reports/*?format=csv` | Confidentiality | Per-report RBAC and anonymized research output |
| CI security artifacts | GitHub Actions artifacts | Supply-chain assurance | CodeQL, SAST, coverage, SBOM, Scorecard |

Open risk: the current repository already tracks `.env` and `secrets/*`. The code now ignores future secret files, but the committed material must be removed from git tracking and rotated before any non-demo deployment.
