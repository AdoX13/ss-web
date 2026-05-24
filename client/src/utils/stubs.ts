// Offline development fixtures (backend handoff §7). Used by unit tests and by
// the review queue's dev-only "sample data" affordance so the UI can be
// exercised without the full Go + MongoDB + MQTT stack running.

import type { ReviewItem } from '../types/review';

export const stubReviewItems: ReviewItem[] = [
  {
    id: '6650000000000000000001',
    image_id: '6650000000000000000010',
    field_name: 'patient_name',
    original_value: 'Popescu Ion',
    original_confidence: 0.71,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
  {
    id: '6650000000000000000002',
    image_id: '6650000000000000000010',
    field_name: 'control_type',
    original_value: null,
    original_confidence: 0,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
  {
    id: '6650000000000000000003',
    image_id: '6650000000000000000011',
    field_name: 'exam_date',
    original_value: '15.03.2024',
    original_confidence: 0.82,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
];
