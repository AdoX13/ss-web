// Live review-queue updates over WebSocket (backend handoff §5).
//
// CONTRACT GAP: the backend authenticates /ws/review with the Authorization
// header (auth.WithAuth). Browsers CANNOT set headers on the WebSocket API, so
// this connection currently cannot authenticate from the browser. We send the
// token as a query parameter (the standard browser-compatible pattern) so this
// hook works the moment the backend accepts it, and we fail silently otherwise.
// The review queue stays correct via polling regardless. See docs ISSUES.md.

import { useEffect, useRef } from 'react';
import { API_BASE_URL, tokenStore } from '../utils/api';
import type { ReviewItem } from '../types/review';

export type SocketStatus = 'connecting' | 'open' | 'closed';

export function useReviewSocket(
  enabled: boolean,
  onItem: (item: ReviewItem) => void,
  onStatus?: (status: SocketStatus) => void,
): void {
  const onItemRef = useRef(onItem);
  onItemRef.current = onItem;
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;

  useEffect(() => {
    if (!enabled) return;
    const token = tokenStore.getAccess();
    if (!token) return;

    const wsBase = API_BASE_URL.replace(/^http/i, 'ws');
    let ws: WebSocket;
    try {
      ws = new WebSocket(`${wsBase}/ws/review?token=${encodeURIComponent(token)}`);
    } catch {
      onStatusRef.current?.('closed');
      return;
    }

    onStatusRef.current?.('connecting');
    ws.onopen = () => onStatusRef.current?.('open');
    ws.onmessage = (ev) => {
      try {
        onItemRef.current(JSON.parse(ev.data) as ReviewItem);
      } catch {
        // Ignore non-JSON frames.
      }
    };
    ws.onclose = () => onStatusRef.current?.('closed');
    ws.onerror = () => onStatusRef.current?.('closed');

    return () => ws.close();
  }, [enabled]);
}
