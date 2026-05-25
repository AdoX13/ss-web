// Report types. The reports API has two shapes:
//   GET /api/v1/reports        -> ReportMeta[]   (lower-case json tags)
//   GET /api/v1/reports/{name} -> ReportResult   (currently capitalised keys,
//                                  because the Go `reports.Result` struct has
//                                  no json tags — normalised in the data hook)

import type { Role } from './auth';

export interface ReportMeta {
  name: string; // slug, e.g. "recent_exams"
  description: string;
  roles?: Role[]; // absent / empty = all roles allowed
}

export interface ReportResult {
  name: string;
  columns: string[];
  rows: Record<string, unknown>[];
}
