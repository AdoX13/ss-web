// Field labels and enum options for the review queue (backend handoff §6).
// `control_type` and `medical_opinion` are enum fields — render a <select>
// when correcting them, not a free-text input. These values match the
// Statistics page categories exactly.

export const FIELD_LABELS: Record<string, string> = {
  patient_name: 'Nume pacient',
  patient_cnp: 'CNP',
  profession: 'Profesie/Funcție',
  workplace: 'Loc de muncă',
  control_type: 'Tip control',
  medical_opinion: 'Aviz medical',
  exam_date: 'Data examinării',
  expiration_date: 'Valabil până la',
  doctor_name: 'Medic',
};

/** Returns the display label for a field name, falling back to the raw name. */
export const fieldLabel = (name: string): string => FIELD_LABELS[name] ?? name;

export const CONTROL_TYPE_OPTIONS = [
  'Angajare',
  'Periodic',
  'Adaptare',
  'Reluare',
  'Supraveghere',
  'Alte',
] as const;

export const MEDICAL_OPINION_OPTIONS = [
  'APT',
  'APT Condiționat',
  'Inapt Temporar',
  'Inapt',
] as const;

// Fields whose corrected value must be chosen from a fixed enum.
export const ENUM_FIELD_OPTIONS: Record<string, readonly string[]> = {
  control_type: CONTROL_TYPE_OPTIONS,
  medical_opinion: MEDICAL_OPINION_OPTIONS,
};

/** True when a field's value is constrained to a fixed set of options. */
export const isEnumField = (name: string): boolean =>
  Object.prototype.hasOwnProperty.call(ENUM_FIELD_OPTIONS, name);

/** PHI fields that must be masked for non-admin roles (backend handoff §8). */
export const PHI_FIELDS = new Set(['patient_cnp']);
