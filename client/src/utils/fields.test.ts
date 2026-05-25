import { describe, it, expect } from 'vitest';
import {
  fieldLabel,
  isEnumField,
  ENUM_FIELD_OPTIONS,
  CONTROL_TYPE_OPTIONS,
  MEDICAL_OPINION_OPTIONS,
} from './fields';

describe('fields', () => {
  it('maps known field names to Romanian labels', () => {
    expect(fieldLabel('patient_name')).toBe('Nume pacient');
    expect(fieldLabel('control_type')).toBe('Tip control');
    expect(fieldLabel('expiration_date')).toBe('Valabil până la');
  });

  it('falls back to the raw name for unknown fields', () => {
    expect(fieldLabel('totally_unknown')).toBe('totally_unknown');
  });

  it('identifies enum fields', () => {
    expect(isEnumField('control_type')).toBe(true);
    expect(isEnumField('medical_opinion')).toBe(true);
    expect(isEnumField('patient_name')).toBe(false);
    expect(isEnumField('toString')).toBe(false); // not fooled by prototype keys
  });

  it('exposes the expected enum option sets', () => {
    expect(ENUM_FIELD_OPTIONS.control_type).toBe(CONTROL_TYPE_OPTIONS);
    expect(MEDICAL_OPINION_OPTIONS).toContain('APT');
    expect(MEDICAL_OPINION_OPTIONS).toContain('Inapt');
    expect(CONTROL_TYPE_OPTIONS).toContain('Periodic');
  });
});
