# Threat Mitigations

| Threat | Control | Implementation |
|---|---|---|
| PHI disclosure from database | Field-level encryption | `server/crypto` AES-256-GCM envelopes; `server/repository/medical_repository.go` encrypted projection |
| CNP lookup leakage | Deterministic keyed digest | HMAC-SHA256 via `crypto.HashCNP`, unique sparse index on `patients.cnp_hash` |
| JWT token forgery | Strong env secret | `JWT_SECRET` loaded from environment; `.env.example` documents random secret |
| MQTT MITM | Mutual TLS | Mosquitto listener 8883 with CA/server cert/client cert |
| OCR worker exploit | Process isolation | `ocr-worker` over Unix socket, no network target, request caps |
| Audit repudiation | Append-only writer | `audit.NewMongoWriter` exposes insert/list only |
| Evidence tampering | Hash chain plus signature | `evidence.MongoChain` computes `prev_hash`, `this_hash`, Ed25519 signature |
| Research re-identification | k-anonymity | R4 suppresses buckets below k=5 and omits direct identifiers |
| Report abuse | RBAC | Per-report `Roles()` filtered in `/api/v1/reports` |
| Schema drift | Mongo validators | `repository.EnsureSchema` installs `$jsonSchema` validators and report indexes |
| Secret leakage | Ignore and rotate | `.gitignore`, `.env.example`, key-management doc |
