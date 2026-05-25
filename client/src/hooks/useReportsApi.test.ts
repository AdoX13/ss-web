import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';

const apiJson = vi.fn();
const apiFetch = vi.fn();
vi.mock('../utils/api', () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
  apiFetch: (...args: unknown[]) => apiFetch(...args),
  ApiError: class ApiError extends Error {
    status: number;
    constructor(status: number, message: string) {
      super(message);
      this.status = status;
    }
  },
}));

import { useReportList, runReport, downloadReportCsv } from './useReports';

beforeEach(() => {
  apiJson.mockReset();
  apiFetch.mockReset();
});

describe('useReportList', () => {
  it('loads the role-filtered report list', async () => {
    apiJson.mockResolvedValue([
      { name: 'recent_exams', description: 'Recent', roles: [] },
    ]);
    const { result } = renderHook(() => useReportList());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.reports).toHaveLength(1);
    expect(result.current.reports[0].name).toBe('recent_exams');
  });
});

describe('runReport', () => {
  it('requests JSON and normalises the capitalised shape', async () => {
    apiJson.mockResolvedValue({ Name: 'r', Columns: ['a'], Rows: [{ a: 1 }] });
    const res = await runReport('r', { from: '2026-01-01T00:00:00Z' });
    expect(res).toEqual({ name: 'r', columns: ['a'], rows: [{ a: 1 }] });
    expect(apiJson).toHaveBeenCalledWith(expect.stringContaining('format=json'));
  });
});

describe('downloadReportCsv', () => {
  it('pulls CSV bytes via apiFetch and triggers a client download', async () => {
    const clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {});
    URL.createObjectURL = vi.fn(
      () => 'blob:x',
    ) as unknown as typeof URL.createObjectURL;
    URL.revokeObjectURL = vi.fn() as unknown as typeof URL.revokeObjectURL;
    apiFetch.mockResolvedValue(
      new Response('a,b\n1,2', {
        status: 200,
        headers: { 'Content-Type': 'text/csv' },
      }),
    );

    await downloadReportCsv('recent_exams', {});

    expect(apiFetch).toHaveBeenCalledWith(expect.stringContaining('format=csv'));
    expect(clickSpy).toHaveBeenCalled();
  });
});
