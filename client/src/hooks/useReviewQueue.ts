// Data hook for the review queue. Loads items for the given filters and
// exposes approve / correct / reject actions. Supports optional polling, which
// is our reliable substitute for the WebSocket live-push (see useReviewSocket).

import { useCallback, useEffect, useState } from 'react';
import { apiJson, apiSend, ApiError } from '../utils/api';
import type { ReviewItem, ReviewQueueFilters } from '../types/review';

const buildQuery = (f: ReviewQueueFilters): string => {
  const p = new URLSearchParams();
  if (f.status) p.set('status', f.status);
  if (f.field_name) p.set('field_name', f.field_name);
  if (f.image_id) p.set('image_id', f.image_id);
  const s = p.toString();
  return s ? `?${s}` : '';
};

export interface UseReviewQueue {
  items: ReviewItem[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
  approve: (id: string) => Promise<void>;
  correct: (id: string, value: string) => Promise<void>;
  reject: (id: string) => Promise<void>;
}

export function useReviewQueue(
  filters: ReviewQueueFilters,
  pollMs = 0,
): UseReviewQueue {
  const [items, setItems] = useState<ReviewItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { status, field_name, image_id } = filters;

  const load = useCallback(async () => {
    setError(null);
    try {
      const query = buildQuery({ status, field_name, image_id });
      const data = await apiJson<ReviewItem[]>(`/api/v1/review-queue${query}`);
      setItems(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : 'Failed to load the review queue.',
      );
    } finally {
      setLoading(false);
    }
  }, [status, field_name, image_id]);

  useEffect(() => {
    setLoading(true);
    void load();
  }, [load]);

  // Optional polling — the dependable fallback for live updates.
  useEffect(() => {
    if (pollMs <= 0) return;
    const id = window.setInterval(() => void load(), pollMs);
    return () => window.clearInterval(id);
  }, [pollMs, load]);

  const act = useCallback(
    async (id: string, action: 'approve' | 'correct' | 'reject', body?: unknown) => {
      await apiSend(`/api/v1/review-queue/${id}/${action}`, {
        method: 'POST',
        headers: body ? { 'Content-Type': 'application/json' } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      });
      await load();
    },
    [load],
  );

  return {
    items,
    loading,
    error,
    refresh: () => {
      setLoading(true);
      void load();
    },
    approve: (id) => act(id, 'approve'),
    correct: (id, value) => act(id, 'correct', { corrected_value: value }),
    reject: (id) => act(id, 'reject'),
  };
}
