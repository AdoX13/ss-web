# STRIDE Threat Table

| Asset | STRIDE | Threat | Impact | Existing or planned control |
|---|---|---|---|---|
| PHI in MongoDB | Information disclosure | Database dump reveals names, CNP, workplace | High | P6 encrypted `patients` and `medical_records`; legacy `photos` protected by RBAC and kept for Lab 1 compatibility |
| PHI in MongoDB | Tampering | Attacker changes medical opinion or expiration date | High | Mongo validators, audit log, evidence chain for review actions |
| JWT secret | Spoofing | Leaked weak JWT secret permits forged tokens | High | Env-based secret, `.env.example`, rotate committed secret |
| JWT secret | Elevation of privilege | Forged role claim grants admin/report access | High | HMAC signing validation, RBAC middleware, strong secret requirement |
| mTLS CA private key | Spoofing | Attacker issues client cert and publishes fake images | High | Keep CA key out of git, rotate CA, Mosquitto `require_certificate` |
| MQTT broker | Information disclosure | Plain MQTT interception of medical images | High | Lab 7 mTLS on port 8883 |
| MQTT broker | Tampering | MITM modifies image payload before OCR | High | mTLS server/client authentication |
| OCR worker | Denial of service | Oversized or malformed images exhaust worker | Medium | 10 MB cap, request timeout, concurrency semaphore |
| OCR worker | Elevation of privilege | Crafted image exploits native OCR dependency | High | Separate worker, Unix socket, gVisor/no-network target |
| Audit trail | Repudiation | Reviewer denies approving or correcting OCR data | Medium | Audit writer records actor/action/resource/timestamp |
| Evidence chain | Tampering | Attacker updates or removes audit evidence | High | Hash chain with Ed25519 signatures and verify endpoint |
| Reports | Information disclosure | Researcher exports identifiable patient data | High | R4 aggregate k-anonymity, no names/CNP in research export |
| Reports | Denial of service | Expensive unbounded report query | Medium | Default limits for row reports and indexed fields |
| CI artifacts | Tampering | Security report altered after pipeline | Medium | Upload artifacts, SBOM, CodeQL/SAST references |
