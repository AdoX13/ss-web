import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';

let token: string | null = 't';
vi.mock('../utils/api', () => ({
  API_BASE_URL: 'http://localhost:8080',
  tokenStore: { getAccess: () => token },
}));

import { useReviewSocket } from './useReviewSocket';

type MsgHandler = ((ev: { data: string }) => void) | null;

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: MsgHandler = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  close() {
    this.closed = true;
  }
}

beforeEach(() => {
  token = 't';
  MockWebSocket.instances = [];
  (globalThis as unknown as { WebSocket: typeof MockWebSocket }).WebSocket =
    MockWebSocket;
});

describe('useReviewSocket', () => {
  it('connects with the token in the query and forwards parsed items', () => {
    const onItem = vi.fn();
    renderHook(() => useReviewSocket(true, onItem));
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain('token=t');
    MockWebSocket.instances[0].onmessage?.({ data: JSON.stringify({ id: 'x' }) });
    expect(onItem).toHaveBeenCalledWith(expect.objectContaining({ id: 'x' }));
  });

  it('is inert when disabled', () => {
    renderHook(() => useReviewSocket(false, vi.fn()));
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it('is inert without a token', () => {
    token = null;
    renderHook(() => useReviewSocket(true, vi.fn()));
    expect(MockWebSocket.instances).toHaveLength(0);
  });
});
