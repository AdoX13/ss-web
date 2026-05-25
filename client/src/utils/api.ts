// Centralised API client for the MedSec-OCR frontend.
//
// Responsibilities (backend handoff §1.5):
//   1. Resolve the backend base URL (VITE_API_BASE_URL or localhost:8080).
//   2. Attach the JWT access token to every authenticated request.
//   3. On a 401, transparently refresh the access token once and retry.
//   4. On a second 401 (or no refresh token), clear the session and redirect
//      to /login.
//
// The backend issues 15-minute access tokens and 7-day refresh tokens with
// rotation, and returns *plain-text* error bodies (net/http `http.Error`).

const DEFAULT_BASE_URL = 'http://127.0.0.1:8080';

const normalize = (url: string) => url.replace(/\/+$/, '');

export const API_BASE_URL = normalize(
  import.meta.env.VITE_API_BASE_URL ?? DEFAULT_BASE_URL,
);

// ── Token storage ────────────────────────────────────────────────────────────
// localStorage keeps the session across reloads. These keys are the single
// source of truth shared between apiFetch and AuthContext.
const ACCESS_KEY = 'access_token';
const REFRESH_KEY = 'refresh_token';
const EMAIL_KEY = 'auth_email';
const ROLE_KEY = 'auth_role';

export interface StoredSession {
  accessToken: string;
  refreshToken: string;
  email: string;
  role: string;
}

export const tokenStore = {
  getAccess: (): string | null => localStorage.getItem(ACCESS_KEY),
  getRefresh: (): string | null => localStorage.getItem(REFRESH_KEY),
  getEmail: (): string | null => localStorage.getItem(EMAIL_KEY),
  getRole: (): string | null => localStorage.getItem(ROLE_KEY),
  setTokens: (access: string, refresh: string) => {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  setSession: (s: StoredSession) => {
    localStorage.setItem(ACCESS_KEY, s.accessToken);
    localStorage.setItem(REFRESH_KEY, s.refreshToken);
    localStorage.setItem(EMAIL_KEY, s.email);
    localStorage.setItem(ROLE_KEY, s.role);
  },
  clear: () => {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(EMAIL_KEY);
    localStorage.removeItem(ROLE_KEY);
    // Drop the legacy skeleton key too, if present.
    localStorage.removeItem('token');
  },
};

const buildUrl = (path: string) => {
  if (path.startsWith('http')) return path;
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${API_BASE_URL}${normalizedPath}`;
};

// Auth endpoints must not trigger the refresh/redirect dance: a 401 there means
// bad credentials or an invalid refresh token, not an expired access token.
const isAuthPath = (path: string) => path.includes('/api/v1/auth/');

const withAuthHeaders = (init?: RequestInit): RequestInit => {
  const token = tokenStore.getAccess();
  const headers = new Headers(init?.headers);
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  return { ...init, headers };
};

// Single-flight refresh: concurrent 401s share one refresh round-trip so we
// never fire two /auth/refresh calls (which would revoke each other's token).
let refreshInFlight: Promise<boolean> | null = null;

const doRefresh = async (): Promise<boolean> => {
  const refreshToken = tokenStore.getRefresh();
  if (!refreshToken) return false;
  try {
    const res = await fetch(buildUrl('/api/v1/auth/refresh'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return false;
    const data = (await res.json()) as {
      access_token: string;
      refresh_token: string;
    };
    tokenStore.setTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
};

const refreshOnce = (): Promise<boolean> => {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
};

const redirectToLogin = () => {
  tokenStore.clear();
  if (window.location.pathname !== '/login') {
    window.location.assign('/login');
  }
};

/**
 * Drop-in `fetch` wrapper. Attaches the bearer token and, on a 401, attempts a
 * single silent refresh + retry before giving up and redirecting to /login.
 */
export const apiFetch = async (
  path: string,
  init?: RequestInit,
): Promise<Response> => {
  const res = await fetch(buildUrl(path), withAuthHeaders(init));

  if (res.status !== 401 || isAuthPath(path)) {
    return res;
  }

  const refreshed = await refreshOnce();
  if (!refreshed) {
    redirectToLogin();
    return res;
  }

  const retry = await fetch(buildUrl(path), withAuthHeaders(init));
  if (retry.status === 401) {
    redirectToLogin();
  }
  return retry;
};

// ── Typed helpers ────────────────────────────────────────────────────────────

/** Error carrying the HTTP status, thrown by apiJson / apiSend on !ok. */
export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

/** Reads an error body as text (the backend returns plain-text errors). */
const errorMessage = async (res: Response): Promise<string> => {
  try {
    const text = (await res.text()).trim();
    return text || `${res.status} ${res.statusText}`;
  } catch {
    return `${res.status} ${res.statusText}`;
  }
};

/** Performs a request and parses JSON, throwing ApiError on a non-2xx status. */
export async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await apiFetch(path, init);
  if (!res.ok) {
    throw new ApiError(res.status, await errorMessage(res));
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

/** Performs a mutating request expecting an empty (204) body. */
export async function apiSend(path: string, init?: RequestInit): Promise<void> {
  const res = await apiFetch(path, init);
  if (!res.ok) {
    throw new ApiError(res.status, await errorMessage(res));
  }
}

/** Convenience POST with a JSON body. */
export const postJson = <T>(path: string, body: unknown): Promise<T> =>
  apiJson<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
