import { describe, it, expect } from 'vitest';
import fc from 'fast-check';
import { normalizeResult } from './useReports';

// Property: whatever the backend throws at us, the normaliser must yield a
// safe ReportResult — name is a string, columns and rows are arrays — so the
// rendering layer never has to guard against undefined.
describe('normalizeResult (property-based)', () => {
  it('always returns a well-formed ReportResult', () => {
    fc.assert(
      fc.property(fc.anything(), (raw) => {
        const r = normalizeResult(raw);
        expect(typeof r.name).toBe('string');
        expect(Array.isArray(r.columns)).toBe(true);
        expect(Array.isArray(r.rows)).toBe(true);
      }),
    );
  });

  it('prefers lower-case keys but accepts capitalised ones', () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.array(fc.string()),
        fc.boolean(),
        (name, columns, capitalised) => {
          const raw = capitalised
            ? { Name: name, Columns: columns }
            : { name, columns };
          const r = normalizeResult(raw);
          expect(r.name).toBe(name);
          expect(r.columns).toEqual(columns);
        },
      ),
    );
  });
});
