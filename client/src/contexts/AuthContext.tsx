// Real JWT authentication context. Replaces the development stub that
// auto-logged-in as a guest admin. Tokens live in localStorage (see
// utils/api.ts `tokenStore`); this context exposes the derived session state
// (email, role, helpers) to the React tree.

import React, {
  createContext,
  useState,
  useContext,
  useEffect,
  useCallback,
} from 'react';
import { apiFetch, tokenStore } from '../utils/api';
import type { LoginResponse, Role } from '../types/auth';
import { isRole, roleSatisfies } from '../types/auth';

interface AuthContextType {
  isLoggedIn: boolean;
  loading: boolean;
  email: string | null;
  role: Role | null;
  isAdmin: boolean;
  /** Access token, exposed for backward compatibility with legacy pages. */
  token: string | null;
  /** True if the current role is one of the provided roles. */
  hasRole: (...roles: Role[]) => boolean;
  login: (data: LoginResponse) => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType>({
  isLoggedIn: false,
  loading: true,
  email: null,
  role: null,
  isAdmin: false,
  token: null,
  hasRole: () => false,
  login: () => {},
  logout: async () => {},
});

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => useContext(AuthContext);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [email, setEmail] = useState<string | null>(null);
  const [role, setRole] = useState<Role | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Restore the session from storage on first load.
  useEffect(() => {
    const access = tokenStore.getAccess();
    const storedRole = tokenStore.getRole();
    if (access && isRole(storedRole)) {
      setToken(access);
      setRole(storedRole);
      setEmail(tokenStore.getEmail());
    }
    setLoading(false);
  }, []);

  const login = useCallback((data: LoginResponse) => {
    tokenStore.setSession({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      email: data.email,
      role: data.role,
    });
    setToken(data.access_token);
    setEmail(data.email);
    setRole(data.role);
  }, []);

  const logout = useCallback(async () => {
    try {
      // Best-effort: revoke refresh tokens server-side (handoff §1.4).
      await apiFetch('/api/v1/auth/logout', { method: 'POST' });
    } catch {
      // Network failure shouldn't trap the user in a logged-in state.
    }
    tokenStore.clear();
    setToken(null);
    setEmail(null);
    setRole(null);
  }, []);

  const hasRole = useCallback(
    (...roles: Role[]) => roleSatisfies(role, roles),
    [role],
  );

  const value: AuthContextType = {
    isLoggedIn: token !== null,
    loading,
    email,
    role,
    isAdmin: role === 'admin',
    token,
    hasRole,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export default AuthContext;
