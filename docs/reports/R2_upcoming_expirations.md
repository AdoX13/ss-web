# R2 Upcoming Expirations

Endpoint: `GET /api/v1/reports/upcoming_expirations`

Roles: `admin`, `doctor`

Query: `photos.data_urm_examinari` in the selected date range. If the selected range is entirely in the past, the backend defaults to the next 30 days.

Columns: `document_id`, `patient`, `profession`, `medical_opinion`, `expires_at`, `days_until`.

Interpretation: worklist for renewals and expiring occupational-medicine clearances.
