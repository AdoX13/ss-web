# R5 OCR Performance

Endpoint: `GET /api/v1/reports/ocr_performance`

Roles: `admin`, `auditor`

Query: `ocr_results.extracted_at` in the selected date range. If the normalized P6 collection is empty, the implementation falls back to Lab 1 `photos.timestamp` for demo compatibility.

Columns: `metric`, `value`.

Metrics: total documents, average OCR confidence, documents routed to review, high-confidence documents, and average processing time when available.

Interpretation: quality gate for the OCR pipeline and review workload forecasting.
