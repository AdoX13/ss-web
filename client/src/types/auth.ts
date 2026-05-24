// Auth-related types shared across the app. Matches the backend contract in
// `server/routes/auth_routes.go` and `server/domain/user.go`.

export type Role = 'admin' | 'doctor' | 'researcher' | 'auditor';

export const ALL_ROLES: Role[] = ['admin', 'doctor', 'researcher', 'auditor'];

// Human-readable labels for roles (used in nav badges, admin dropdowns, etc.).
export const ROLE_LABELS: Record<Role, string> = {
  admin: 'Administrator',
  doctor: 'Doctor',
  researcher: 'Researcher',
  auditor: 'Auditor',
};

// Response shape of POST /api/v1/auth/login.
export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  email: string;
  role: Role;
}

export const isRole = (value: unknown): value is Role =>
  typeof value === 'string' && (ALL_ROLES as string[]).includes(value);

// Pure core of the RBAC checks (AuthContext.hasRole, ProtectedRoute). True when
// `role` is set and present in `allowed`.
export const roleSatisfies = (role: Role | null, allowed: Role[]): boolean =>
  role !== null && allowed.includes(role);
