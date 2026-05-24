// Reports data access. Handles the two API shapes:
//   - list endpoint returns lower-case ReportMeta[]
//   - run endpoint returns a Go struct currently serialised with CAPITALISED
//     keys ({Name, Columns, Rows}); normalizeResult tolerates both so the UI
//     keeps working when the backend later adds json tags.

import { useEffect, useState } from 'react';
import { apiJson, apiFetch, ApiError } from '../utils/api';
import type { ReportMeta, ReportResult } from '../types/report';

export function normalizeResult(raw: unknown): ReportResult {
  const r = (raw ?? {}) as Record<string, unknown>;
  const columns = (r.columns ?? r.Columns ?? []) as unknown;
  const rows = (r.rows ?? r.Rows ?? []) as unknown;
  const name = (r.name ?? r.Name ?? '') as string;
  return {
    name: typeof name === 'string' ? name : '',
    columns: Array.isArray(columns) ? (columns as string[]) : [],
    rows: Array.isArray(rows) ? (rows as Record<string, unknown>[]) : [],
  };
}

export interface ReportParams {
  from?: string; // RFC3339
  to?: string; // RFC3339
}

const buildParams = (p: ReportParams, extra?: Record<string, string>): string => {
  const q = new URLSearchParams();
  if (p.from) q.set('from', p.from);
  if (p.to) q.set('to', p.to);
  if (extra) for (const [k, v] of Object.entries(extra)) q.set(k, v);
  const s = q.toString();
  return s ? `?${s}` : '';
};

export function useReportList() {
  const [reports, setReports] = useState<ReportMeta[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const data = await apiJson<ReportMeta[]>('/api/v1/reports');
        if (active) setReports(Array.isArray(data) ? data : []);
      } catch (e) {
        if (active) {
          setError(e instanceof ApiError ? e.message : 'Failed to load reports.');
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  return { reports, loading, error };
}

export async function runReport(
  name: string,
  p: ReportParams,
): Promise<ReportResult> {
  const raw = await apiJson<unknown>(
    `/api/v1/reports/${encodeURIComponent(name)}${buildParams(p, { format: 'json' })}`,
  );
  return normalizeResult(raw);
}

// CSV export must go through apiFetch (the bearer token can't ride on a plain
// <a download>), so we pull the bytes and trigger a client-side download.
export async function downloadReportCsv(
  name: string,
  p: ReportParams,
): Promise<void> {
  const res = await apiFetch(
    `/api/v1/reports/${encodeURIComponent(name)}${buildParams(p, { format: 'csv' })}`,
  );
  if (!res.ok) {
    throw new ApiError(res.status, (await res.text()).trim() || 'Export failed.');
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${name}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

/** Heuristic for the "not yet implemented" stub rows the backend returns. */
export const isStubResult = (r: ReportResult): boolean =>
  r.rows.length === 1 &&
  'status' in r.rows[0] &&
  r.rows[0].status === 'not yet implemented';
