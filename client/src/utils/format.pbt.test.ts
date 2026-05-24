import { describe, it, expect } from 'vitest';
import fc from 'fast-check';
import { timeAgo, formatDate, formatDateTime } from './format';

const NOW = Date.parse('2026-05-24T12:00:00Z');

describe('timeAgo (property-based)', () => {
  it('reports seconds for deltas under a minute', () => {
    fc.assert(
      fc.property(fc.integer({ min: 0, max: 59 }), (s) => {
        const iso = new Date(NOW - s * 1000).toISOString();
        expect(timeAgo(iso, NOW)).toBe(`${s}s ago`);
      }),
    );
  });

  it('reports minutes for deltas of 1–59 minutes', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 59 }), (m) => {
        const iso = new Date(NOW - m * 60_000).toISOString();
        expect(timeAgo(iso, NOW)).toBe(`${m}m ago`);
      }),
    );
  });

  it('reports hours for deltas of 1–23 hours', () => {
    fc.assert(
      fc.property(fc.integer({ min: 1, max: 23 }), (h) => {
        const iso = new Date(NOW - h * 3_600_000).toISOString();
        expect(timeAgo(iso, NOW)).toBe(`${h}h ago`);
      }),
    );
  });

  it('always returns a string and never throws for any finite delta', () => {
    fc.assert(
      fc.property(fc.integer({ min: 0, max: 10 ** 12 }), (delta) => {
        const iso = new Date(NOW - delta).toISOString();
        expect(typeof timeAgo(iso, NOW)).toBe('string');
      }),
    );
  });

  it('returns an em dash for missing or invalid input', () => {
    expect(timeAgo(undefined)).toBe('—');
    expect(timeAgo(null)).toBe('—');
    expect(timeAgo('not-a-date')).toBe('—');
  });
});

describe('formatDate / formatDateTime (property-based)', () => {
  it('never throws for arbitrary strings', () => {
    fc.assert(
      fc.property(fc.string(), (s) => {
        expect(typeof formatDate(s)).toBe('string');
        expect(typeof formatDateTime(s)).toBe('string');
      }),
    );
  });

  it('returns an em dash for nullish input', () => {
    expect(formatDate(undefined)).toBe('—');
    expect(formatDateTime(null)).toBe('—');
  });
});
