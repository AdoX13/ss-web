# R3 Compliance Percentage

Endpoint: `GET /api/v1/reports/compliance_percentage`

Roles: `admin`, `doctor`, `auditor`

Query: documents up to `to`. A worker is counted as valid when the medical opinion is APT or APT Conditionat and the expiration date is not before `to`.

Columns: `status`, `count`, `percentage`.

Interpretation: aggregate compliance signal. It intentionally avoids patient-level PHI, so auditors may run it.
