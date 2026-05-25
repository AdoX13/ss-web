# Security Incident Response Runbook

## Overview
This runbook defines the procedures for responding to security incidents in the MedSec-OCR environment. Given the processing of sensitive medical data (PHI), swift and coordinated responses are critical.

## Severity Levels

| Level | Description | Examples |
|-------|-------------|----------|
| **P1** | **Critical Breach** | Unauthorized access to MongoDB, leaked certificates, OCR sandbox breakout, PHI exposure. |
| **P2** | **High Impact** | Unauthorized access to a single account, suspicious MQTT payloads, DAST/SAST critical findings in production. |
| **P3** | **Medium Impact** | Unpatched dependencies with known CVEs, rate-limiting triggers, failed authentication spikes. |
| **P4** | **Informational** | Expected security scanner noise, minor configuration drift. |

## Response Procedures

### P1: Critical Breach (Data Exposure or System Compromise)
1. **Containment**: 
   - Immediately stop the MQTT Broker and Go API to prevent further data ingestion or exfiltration:
     `docker compose stop broker go-api`
   - Disconnect the network if necessary.
2. **Revocation**:
   - If certificates are compromised, rotate the CA and all client certificates.
3. **Forensics**:
   - Preserve logs (`.dev-runtime/`, `docker compose logs`).
   - Do not restart services until the root cause is identified.
4. **Notification**:
   - Notify the security team and data protection officer (DPO).

### P2: High Impact (Unauthorized Access)
1. **Containment**:
   - Lock out the affected user account (change role to `suspended` in MongoDB).
   - If an IoT device is compromised, revoke its certificate or block its `DEVICE_ID`.
2. **Investigation**:
   - Review audit logs for actions performed by the compromised account.
3. **Remediation**:
   - Force password reset, rotate JWT secrets if applicable.

### P3: Medium Impact (Vulnerabilities)
1. **Investigation**:
   - Verify the SAST/SCA finding.
2. **Remediation**:
   - Apply patches or update dependencies within the SLA (typically 7-14 days).
   - Re-run the CI pipeline to verify the fix.

## Post-Mortem Template
Every P1 and P2 incident requires a post-mortem document covering:
1. **Timeline**: Exact sequence of events.
2. **Root Cause**: How did the attacker gain access?
3. **Impact**: What data was accessed or altered?
4. **Action Items**: Steps to prevent recurrence (e.g., "Add stricter MQTT ACLs").

## Contacts
- **Security Lead**: [Name / Contact Info]
- **Infrastructure Lead**: [Name / Contact Info]
- **DPO**: [Name / Contact Info]
