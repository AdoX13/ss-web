import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import { tokenStore } from '../utils/api';

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <AuthProvider>{children}</AuthProvider>
);

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('AuthContext', () => {
  it('restores a stored session on mount', async () => {
    tokenStore.setSession({
      accessToken: 'A',
      refreshToken: 'R',
      email: 'doc@x',
      role: 'doctor',
    });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.isLoggedIn).toBe(true);
    expect(result.current.role).toBe('doctor');
    expect(result.current.email).toBe('doc@x');
  });

  it('login stores the session and exposes role helpers', async () => {
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    act(() =>
      result.current.login({
        access_token: 'A',
        refresh_token: 'R',
        token_type: 'Bearer',
        email: 'a@x',
        role: 'admin',
      }),
    );
    expect(result.current.isLoggedIn).toBe(true);
    expect(result.current.isAdmin).toBe(true);
    expect(result.current.hasRole('admin')).toBe(true);
    expect(result.current.hasRole('doctor')).toBe(false);
    expect(tokenStore.getAccess()).toBe('A');
  });

  it('logout clears the session (best-effort server call)', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    tokenStore.setSession({
      accessToken: 'A',
      refreshToken: 'R',
      email: 'a@x',
      role: 'admin',
    });
    const { result } = renderHook(() => useAuth(), { wrapper });
    await waitFor(() => expect(result.current.isLoggedIn).toBe(true));
    await act(async () => {
      await result.current.logout();
    });
    expect(result.current.isLoggedIn).toBe(false);
    expect(tokenStore.getAccess()).toBeNull();
  });
});
