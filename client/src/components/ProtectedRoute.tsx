import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import type { Role } from '../types/auth';
import AccessDenied from './AccessDenied';

interface ProtectedRouteProps {
  // When true (default), unauthenticated users are sent to /login. When false,
  // authenticated users are bounced away (used to guard /login and /register).
  authRequired?: boolean;
  // Optional RBAC: if set, the user's role must be one of these.
  roles?: Role[];
}

const Spinner = () => (
  <div className="flex justify-center items-center min-h-[60vh]">
    <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500" />
  </div>
);

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  authRequired = true,
  roles,
}) => {
  const { isLoggedIn, loading, hasRole } = useAuth();

  if (loading) return <Spinner />;

  if (authRequired && !isLoggedIn) {
    return <Navigate to="/login" replace />;
  }
  if (!authRequired && isLoggedIn) {
    return <Navigate to="/" replace />;
  }
  if (roles && roles.length > 0 && !hasRole(...roles)) {
    return <AccessDenied requiredRoles={roles} />;
  }

  return <Outlet />;
};

export default ProtectedRoute;
