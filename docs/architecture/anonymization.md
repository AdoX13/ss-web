# Anonymization Design

R4 (`anonymized_export`) is designed for the researcher role. It does not return names, CNP values, image identifiers, exact dates, addresses, or workplace text.

Current implementation:

| Field | Treatment |
|---|---|
| Patient name | Removed |
| CNP | Removed from export |
| Exam date | Reduced to `YYYY-MM` |
| Profession | Kept only inside buckets with k >= 5 |
| Control type | Aggregated set |
| Medical opinion | Aggregated set |
| Small buckets | Suppressed into a single `suppressed` bucket |

k-anonymity threshold: `k = 5`.

Residual limitation: the current Lab 1 schema does not collect age band or postcode prefix. Once `patients` contains encrypted DOB and address-derived postcode prefix, the quasi-identifier bucket should become `(age_band, postcode_prefix, profession, exam_month)`.
