import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';

const apiJson = vi.fn();
const apiSend = vi.fn();
vi.mock('../utils/api', () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
  apiSend: (...args: unknown[]) => apiSend(...args),
  ApiError: class ApiError extends Error {
    status = 0;
  },
}));

import { useReviewQueue } from './useReviewQueue';

const items = [
  {
    id: 'a1',
    image_id: 'i1',
    field_name: 'patient_name',
    original_value: 'X',
    original_confidence: 0.7,
    status: 'pending',
    created_at: '2026-01-01T00:00:00Z',
  },
];

beforeEach(() => {
  apiJson.mockReset();
  apiSend.mockReset();
  apiJson.mockResolvedValue(items);
  apiSend.mockResolvedValue(undefined);
});

describe('useReviewQueue', () => {
  it('loads items for the given filters', async () => {
    const { result } = renderHook(() => useReviewQueue({ status: 'pending' }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.items).toEqual(items);
    expect(apiJson).toHaveBeenCalledWith(expect.stringContaining('status=pending'));
  });

  it('approve POSTs to the approve endpoint then reloads', async () => {
    const { result } = renderHook(() => useReviewQueue({ status: 'pending' }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    await act(async () => {
      await result.current.approve('a1');
    });
    expect(apiSend).toHaveBeenCalledWith(
      expect.stringContaining('/a1/approve'),
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('correct sends the corrected value in the body', async () => {
    const { result } = renderHook(() => useReviewQueue({ status: 'pending' }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    await act(async () => {
      await result.current.correct('a1', 'APT');
    });
    const call = apiSend.mock.calls.find((c) => String(c[0]).includes('/correct'));
    expect(call).toBeTruthy();
    expect(JSON.parse((call![1] as RequestInit).body as string)).toEqual({
      corrected_value: 'APT',
    });
  });

  it('surfaces load errors', async () => {
    apiJson.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderHook(() => useReviewQueue({ status: 'pending' }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBeTruthy();
  });
});
