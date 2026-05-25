// Review-queue types. Mirrors `server/domain/review_item.go`.
// NOTE: `reviewer_id` from the original handoff was renamed to `reviewer_email`.

export type ReviewItemStatus = 'pending' | 'approved' | 'corrected' | 'rejected';

export const REVIEW_STATUSES: ReviewItemStatus[] = [
  'pending',
  'approved',
  'corrected',
  'rejected',
];

export interface ReviewItem {
  id: string;
  image_id: string;
  field_name: string;
  original_value: string | null;
  original_confidence: number; // 0.0 – 1.0
  status: ReviewItemStatus;
  reviewer_email?: string;
  corrected_value?: string;
  reviewed_at?: string;
  created_at: string;
}

export interface ReviewQueueFilters {
  status?: ReviewItemStatus;
  field_name?: string;
  image_id?: string;
}
