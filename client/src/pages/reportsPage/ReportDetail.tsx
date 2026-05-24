import React, { useCallback, useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import {
  runReport,
  downloadReportCsv,
  isStubResult,
} from '../../hooks/useReports';
import type { ReportResult } from '../../types/report';
import { ApiError } from '../../utils/api';
import {
  card,
  heading,
  label,
  input,
  subtleText,
  errorBanner,
  tableHeadCell,
  tableCell,
} from '../../theme/styles';

const todayISO = () => new Date().toISOString().slice(0, 10);
const daysAgoISO = (n: number) => {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().slice(0, 10);
};

// Convert a yyyy-mm-dd input to an RFC3339 timestamp (what the backend parses).
const toRFC3339 = (date: string, endOfDay = false): string => {
  const d = new Date(`${date}T${endOfDay ? '23:59:59' : '00:00:00'}Z`);
  return d.toISOString();
};

const cell = (v: unknown): string => {
  if (v === null || v === undefined) return '';
  if (typeof v === 'object') return JSON.stringify(v);
  return String(v);
};

// A report is auto-chartable when it has rows shaped like (label, number).
const chartableData = (
  result: ReportResult,
): { name: string; value: number }[] | null => {
  if (result.columns.length !== 2 || result.rows.length === 0) return null;
  const [labelCol, valueCol] = result.columns;
  const data: { name: string; value: number }[] = [];
  for (const row of result.rows) {
    const v = row[valueCol];
    const num = typeof v === 'number' ? v : Number(v);
    if (Number.isNaN(num)) return null;
    data.push({ name: cell(row[labelCol]), value: num });
  }
  return data;
};

const ReportDetailPage: React.FC = () => {
  const { name = '' } = useParams();
  const [from, setFrom] = useState(daysAgoISO(30));
  const [to, setTo] = useState(todayISO());
  const [result, setResult] = useState<ReportResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const params = useCallback(
    () => ({ from: toRFC3339(from), to: toRFC3339(to, true) }),
    [from, to],
  );

  const run = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setResult(await runReport(name, params()));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to run the report.');
      setResult(null);
    } finally {
      setLoading(false);
    }
  }, [name, params]);

  useEffect(() => {
    void run();
    // Re-run when the report changes; date changes are applied via "Run".
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [name]);

  const onExport = async () => {
    setExporting(true);
    setError(null);
    try {
      await downloadReportCsv(name, params());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'CSV export failed.');
    } finally {
      setExporting(false);
    }
  };

  const stub = result ? isStubResult(result) : false;
  const chart = result && !stub ? chartableData(result) : null;

  return (
    <div className="container mx-auto pb-10">
      <div className="flex items-center gap-2 mb-2">
        <Link
          to="/reports"
          className="text-sky-600 dark:text-sky-400 hover:underline text-sm"
        >
          ← Reports
        </Link>
      </div>
      <h1 className={`${heading} mb-4`}>{result?.name || name}</h1>

      {/* Date range + actions */}
      <div className={`${card} p-4 mb-6 flex flex-wrap items-end gap-4`}>
        <div>
          <label htmlFor="from" className={label}>
            From
          </label>
          <input
            id="from"
            type="date"
            className={input}
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </div>
        <div>
          <label htmlFor="to" className={label}>
            To
          </label>
          <input
            id="to"
            type="date"
            className={input}
            value={to}
            onChange={(e) => setTo(e.target.value)}
          />
        </div>
        <button
          onClick={() => void run()}
          disabled={loading}
          className="px-4 py-2 bg-sky-600 text-white rounded-md hover:bg-sky-700 transition-colors disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-sky-400"
        >
          {loading ? 'Running…' : 'Run'}
        </button>
        <button
          onClick={() => void onExport()}
          disabled={exporting || loading}
          className="px-4 py-2 bg-emerald-600 text-white rounded-md hover:bg-emerald-700 transition-colors disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-emerald-400"
        >
          {exporting ? 'Exporting…' : 'Export CSV'}
        </button>
      </div>

      {error && (
        <div className={`mb-4 ${errorBanner}`} role="alert">
          {error}
        </div>
      )}

      {stub && (
        <div className="mb-4 text-sm flex items-center gap-2 text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md px-3 py-2">
          This report returns placeholder data until the backend aggregation is
          wired up. The layout below will populate automatically.
        </div>
      )}

      {loading ? (
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500" />
        </div>
      ) : result && result.rows.length > 0 ? (
        <div className="space-y-6">
          {chart && (
            <div className={`${card} p-6`}>
              <div className="h-[300px]">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={chart}>
                    <CartesianGrid strokeDasharray="3 3" stroke="currentColor" opacity={0.15} />
                    <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                    <YAxis tick={{ fontSize: 12 }} />
                    <Tooltip />
                    <Bar dataKey="value" fill="#0ea5e9" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}

          {/* Table (also serves as the accessible fallback for the chart) */}
          <div className={`${card} overflow-x-auto`}>
            <table className="min-w-full">
              <caption className="sr-only">{result.name} results</caption>
              <thead className="bg-gray-50 dark:bg-gray-700/50">
                <tr>
                  {result.columns.map((c) => (
                    <th key={c} scope="col" className={tableHeadCell}>
                      {c}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row, i) => (
                  <tr key={i}>
                    {result.columns.map((c) => (
                      <td key={c} className={tableCell}>
                        {cell(row[c])}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className={`${card} text-center py-12 ${subtleText}`}>
          No data for the selected range.
        </div>
      )}
    </div>
  );
};

export default ReportDetailPage;
