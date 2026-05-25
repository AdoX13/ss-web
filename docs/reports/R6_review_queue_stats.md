# R6 Review Queue Stats

Endpoint: `GET /api/v1/reports/review_queue_stats`

Roles: `admin`, `doctor`, `auditor`

Query: `review_items.created_at` in the selected date range.

Columns: `metric`, `value`.

Metrics: pending, approved, corrected, rejected, and average resolution hours.

Interpretation: operational throughput report for human review.
