import { describe, it, expect } from 'vitest';
import { normalizeResult, isStubResult } from './useReports';

describe('normalizeResult', () => {
  it('handles the run endpoint capitalised keys', () => {
    const r = normalizeResult({
      Name: 'recent_exams',
      Columns: ['a', 'b'],
      Rows: [{ a: 1, b: 2 }],
    });
    expect(r).toEqual({
      name: 'recent_exams',
      columns: ['a', 'b'],
      rows: [{ a: 1, b: 2 }],
    });
  });

  it('handles lower-case keys', () => {
    const r = normalizeResult({ name: 'x', columns: ['c'], rows: [] });
    expect(r.name).toBe('x');
    expect(r.columns).toEqual(['c']);
    expect(r.rows).toEqual([]);
  });

  it('defaults missing or nullish input to safe empty values', () => {
    expect(normalizeResult({})).toEqual({ name: '', columns: [], rows: [] });
    expect(normalizeResult(null)).toEqual({ name: '', columns: [], rows: [] });
    expect(normalizeResult(undefined)).toEqual({
      name: '',
      columns: [],
      rows: [],
    });
  });
});

describe('isStubResult', () => {
  it('detects the backend placeholder rows', () => {
    expect(
      isStubResult({
        name: 'recent_exams',
        columns: ['status'],
        rows: [{ status: 'not yet implemented' }],
      }),
    ).toBe(true);
  });

  it('is false for real data and for empty results', () => {
    expect(
      isStubResult({ name: 'x', columns: ['a'], rows: [{ a: 1 }] }),
    ).toBe(false);
    expect(isStubResult({ name: 'x', columns: [], rows: [] })).toBe(false);
  });
});
