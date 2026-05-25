# MISRA/CERT Justification

The course rubric references MISRA/CERT secure-coding evidence. MISRA-C and CERT-C apply to C/C++ codebases. This project has no owned C or C++ application code; the implemented platform is Go, TypeScript/React, Python scripts, Docker Compose, and MongoDB.

Equivalent controls for this stack:

| Layer | Secure-coding gate | Purpose |
|---|---|---|
| Go API and OCR worker | `go test`, fuzz tests, `golangci-lint`, `gosec`, CodeQL | Type safety, race-prone patterns, injection, weak crypto, unsafe file/path handling |
| TypeScript/React | ESLint, TypeScript strict build, fast-check property tests, CodeQL | DOM/XSS risks, unsafe client logic, route guard regressions |
| Supply chain | Dependabot, SBOM CycloneDX/Syft, OpenSSF Scorecard | Vulnerable dependencies and repository hygiene |
| Runtime security | mTLS, Docker secrets, gVisor target for OCR worker | Service and transport hardening |

Mapping to CERT intent:

| CERT intent | Project equivalent |
|---|---|
| Validate input | OCR schema validation, report date parsing, MongoDB validators |
| Use approved cryptography | AES-256-GCM, HMAC-SHA256, Ed25519 from Go standard library |
| Protect credentials | Env-only secrets, `.env.example`, key-management doc |
| Detect tampering | Evidence hash chain and signed records |
| Prevent common web flaws | RBAC middleware, secure headers, auth rate limit, CodeQL/ESLint |

Conclusion: MISRA-C is not applicable because there is no C/C++ code owned by the project. The submitted equivalent is the language-appropriate secure-coding pipeline and the P6 security implementation described above.
