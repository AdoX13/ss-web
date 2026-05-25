# R4 Anonymized Research Export

Endpoint: `GET /api/v1/reports/anonymized_export`

Roles: `researcher`, `admin`

Query: exams in the selected date range.

Columns: `profession`, `exam_month`, `documents`, `control_types`, `medical_opinions`.

Anonymization: direct identifiers are removed, dates are month-only, and buckets with fewer than 5 records are suppressed into a single `suppressed` row.

Interpretation: safe aggregate export for research and statistics. It is not intended for clinical operations.
