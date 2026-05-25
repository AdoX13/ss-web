# Self-Fixed Vulnerabilities

| ID | Severity | Vulnerability | Fix | Residual action |
|---|---|---|---|---|
| P6-001 | High | Local `.env` and certificate material were tracked by git. | Added `.gitignore`, `.env.example`, key/cert generation scripts, and key-management rotation guidance. | Run a reviewed cleanup commit with `git rm --cached .env secrets/*` and rotate material before sharing. |
| P6-002 | High | Legacy `/photos`, `/devices`, and raw `/uploads` PHI paths were reachable without JWT/RBAC. | Wrapped routes with JWT auth and admin/doctor role checks. | Frontend statistics should use report endpoints for researcher-safe data. |
| P6-003 | Medium | Report registry returned placeholder rows instead of real security/reporting logic. | Implemented R1-R6 report logic, role filtering, CSV export, audit, and evidence logging. | Add full Mongo fixture integration tests when local Go/Mongo test tooling is available. |
| P6-004 | High | MQTT compose setup still exposed plaintext 1883 and did not mount broker cert secrets. | Docker Compose now exposes only 8883, mounts broker mTLS secrets, and Mosquitto rejects anonymous clients. | Regenerate local certs with `scripts/gen_certs.sh`. |
| P6-005 | Medium | PHI existed only in the Lab 1 flat `photos` schema. | Added encrypted `patients` and `medical_records` projection with AES-GCM and CNP HMAC lookup. | Migrate historical `photos` with a controlled maintenance job if needed. |
| P6-006 | Medium | Seed data did not exercise encrypted P6 collections. | Extended `scripts/seed_data.py` to seed encrypted medical projection when `MEDSEC_MASTER_KEY` is present. | Install Python `cryptography` for encrypted seeding. |
