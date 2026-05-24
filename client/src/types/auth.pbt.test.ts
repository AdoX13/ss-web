import { describe, it, expect } from 'vitest';
import fc from 'fast-check';
import { isRole, roleSatisfies, ALL_ROLES } from './auth';

describe('isRole (property-based)', () => {
  it('accepts exactly the four known roles', () => {
    fc.assert(
      fc.property(fc.string(), (s) => {
        expect(isRole(s)).toBe((ALL_ROLES as string[]).includes(s));
      }),
    );
  });

  it('rejects non-string values', () => {
    fc.assert(
      fc.property(
        fc.oneof(fc.integer(), fc.boolean(), fc.constant(null), fc.object()),
        (v) => {
          expect(isRole(v)).toBe(false);
        },
      ),
    );
  });
});

describe('roleSatisfies (RBAC guard, property-based)', () => {
  it('is false whenever the role is null', () => {
    fc.assert(
      fc.property(fc.array(fc.constantFrom(...ALL_ROLES)), (allowed) => {
        expect(roleSatisfies(null, allowed)).toBe(false);
      }),
    );
  });

  it('matches set membership of the role in the allowed list', () => {
    fc.assert(
      fc.property(
        fc.constantFrom(...ALL_ROLES),
        fc.array(fc.constantFrom(...ALL_ROLES)),
        (role, allowed) => {
          expect(roleSatisfies(role, allowed)).toBe(allowed.includes(role));
        },
      ),
    );
  });

  it('is false for an empty allow-list', () => {
    fc.assert(
      fc.property(fc.constantFrom(...ALL_ROLES), (role) => {
        expect(roleSatisfies(role, [])).toBe(false);
      }),
    );
  });
});
