# P6 Security Writeup

## Implemented controls

- TARA package covers assets, STRIDE threats, attack trees, and mitigation mapping.
- Field encryption uses AES-256-GCM envelopes with HKDF-derived per-field keys.
- CNP equality lookup uses deterministic HMAC-SHA256, not plaintext CNP.
- Evidence records are hash-chained and Ed25519-signed.
- Audit logs are append-only through the Go writer API and cover review, user-admin, report, and legacy photo access paths.
- MongoDB bootstrap installs validators and query indexes for P6 collections.
- Reports R1-R6 are implemented with RBAC, CSV export, anonymized R4 output, and report execution audit/evidence records.
- Legacy photo/device routes and raw upload serving are protected by JWT plus clinical RBAC.
- MQTT is configured for mTLS on 8883; plaintext 1883 is no longer exposed by Compose.

## SBOM and SAST/SCA

The current server CycloneDX SBOM is `server/sbom.cdx.json`. P6 consumes final CI artifacts from P5 for:

- `golangci-lint`
- `gosec`
- CodeQL
- dependency/SCA scan output
- coverage summary
- ZAP baseline output

No linter suppressions or security-rule bypasses were added by P6. Local verification was limited by the workstation state: `go`, `gofmt`, Docker daemon, and frontend `node_modules` were unavailable during implementation, so final green gates must be confirmed in CI or on a machine with the full toolchain.

## Anonymization

R4 removes names, CNP, document IDs, exact dates, addresses, and workplace. It emits profession/month aggregate buckets only when `k >= 5`; smaller buckets are suppressed into one row. See `docs/architecture/anonymization.md`.

## Evidence and audit verification

Admins and auditors can call:

```text
GET /api/v1/audit
GET /api/v1/evidence/chain/verify
```

The evidence verifier walks sequence order, recomputes `prev_hash`/`this_hash`, and validates every Ed25519 signature.

## Open items before submission

- Run `git rm --cached .env secrets/*` in a reviewed cleanup commit, then rotate the leaked demo values.
- Attach final ZAP/Burp outputs to `docs/security/pentest_report.md`.
- Attach CI SAST/SCA and coverage artifacts from P5.
- Generate the final evaluation PDF from this writeup plus the team development/user docs.
