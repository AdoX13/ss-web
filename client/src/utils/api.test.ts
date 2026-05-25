import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch, apiJson, ApiError, tokenStore } from './api';

const jsonRes = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
const textRes = (text: string, status: number) => new Response(text, { status });

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('apiFetch', () => {
  it('attaches the bearer token from storage', async () => {
    tokenStore.setSession({
      accessToken: 'A',
      refreshToken: 'R',
      email: 'e',
      role: 'admin',
    });
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(jsonRes({ ok: true }));

    await apiFetch('/api/v1/review-queue');

    const init = fetchMock.mock.calls[0][1]!;
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer A');
  });

  it('refreshes once on 401 and retries with the new token', async () => {
    tokenStore.setSession({
      accessToken: 'OLD',
      refreshToken: 'R',
      email: 'e',
      role: 'admin',
    });
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(textRes('unauthorized', 401)) // original
      .mockResolvedValueOnce(jsonRes({ access_token: 'NEW', refresh_token: 'R2' })) // refresh
      .mockResolvedValueOnce(jsonRes({ data: 1 })); // retry

    const res = await apiFetch('/api/v1/review-queue');

    expect(res.status).toBe(200);
    expect(tokenStore.getAccess()).toBe('NEW');
    // The retry (3rd call) used the refreshed token.
    const retryInit = fetchMock.mock.calls[2][1]!;
    expect(new Headers(retryInit.headers).get('Authorization')).toBe('Bearer NEW');
  });

  it('does not attempt a refresh for auth endpoints', async () => {
    tokenStore.setSession({
      accessToken: 'A',
      refreshToken: 'R',
      email: 'e',
      role: 'admin',
    });
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(textRes('bad credentials', 401));

    const res = await apiFetch('/api/v1/auth/login', { method: 'POST' });

    expect(res.status).toBe(401);
    expect(fetchMock).toHaveBeenCalledTimes(1); // no refresh round-trip
  });
});

describe('apiJson', () => {
  it('parses JSON on success', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonRes({ a: 1 }));
    await expect(apiJson('/x')).resolves.toEqual({ a: 1 });
  });

  it('throws ApiError carrying the plain-text body on failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(textRes('boom', 500));
    await expect(apiJson('/x')).rejects.toBeInstanceOf(ApiError);
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(textRes('boom', 500));
    await expect(apiJson('/x')).rejects.toMatchObject({
      status: 500,
      message: 'boom',
    });
  });

  it('returns undefined for a 204 response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }));
    await expect(apiJson('/x')).resolves.toBeUndefined();
  });
});
