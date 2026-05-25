# Reports Overview

P6 implements six role-scoped reports behind `/api/v1/reports` and `/api/v1/reports/{name}`.

| ID | Name | Roles | Primary collection |
|---|---|---|---|
| R1 | `recent_exams` | admin, doctor | `photos` |
| R2 | `upcoming_expirations` | admin, doctor | `photos` |
| R3 | `compliance_percentage` | admin, doctor, auditor | `photos` |
| R4 | `anonymized_export` | researcher, admin | `photos` |
| R5 | `ocr_performance` | admin, auditor | `ocr_results` |
| R6 | `review_queue_stats` | admin, doctor, auditor | `review_items` |

All report executions are audited as `report_run`; the evidence chain also records report name, row count, and requested date window.

## Parameters

`from` and `to` are optional RFC3339 timestamps. If absent, the route defaults to the previous month. `format=csv` streams a CSV response using the report's declared column order.

## Demo data

Run `scripts/seed_data.py` after starting MongoDB. If `MEDSEC_MASTER_KEY` is set and Python package `cryptography` is installed, the script also inserts encrypted `patients`, `medical_records`, `ocr_results`, and `review_items` demo rows.
