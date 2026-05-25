# R1 Recent Exams

Endpoint: `GET /api/v1/reports/recent_exams`

Roles: `admin`, `doctor`

Query: `photos.data` between `from` and `to`.

Columns: `document_id`, `patient`, `profession`, `control_type`, `medical_opinion`, `exam_date`, `expires_at`, `confidence`.

Interpretation: operational list for clinicians to review exams completed during the selected window. Contains PHI, so it is not available to researchers or auditors.
