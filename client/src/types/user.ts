// Admin user-management types. Mirrors `server/domain/user.go` (password
// redacted server-side before serialisation).

import type { Role } from './auth';

export interface User {
  email: string;
  role: Role;
  active: boolean;
}
