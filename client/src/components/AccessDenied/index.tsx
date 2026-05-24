import React from 'react';
import { Link } from 'react-router-dom';
import type { Role } from '../../types/auth';
import { ROLE_LABELS } from '../../types/auth';

interface AccessDeniedProps {
  requiredRoles?: Role[];
}

// Shown when an authenticated user lacks the role for a route. RBAC is also
// enforced server-side (403); this is the friendly client-side counterpart.
const AccessDenied: React.FC<AccessDeniedProps> = ({ requiredRoles }) => (
  <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-4">
    <div className="rounded-full bg-red-100 dark:bg-red-900/30 p-4 mb-6">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-10 w-10 text-red-600 dark:text-red-400"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
        />
      </svg>
    </div>
    <h1 className="text-2xl font-semibold text-gray-800 dark:text-gray-100 mb-2">
      Access denied
    </h1>
    <p className="text-gray-600 dark:text-gray-400 max-w-md mb-6">
      You don&rsquo;t have permission to view this page.
      {requiredRoles && requiredRoles.length > 0
        ? ` Requires: ${requiredRoles.map((r) => ROLE_LABELS[r]).join(' or ')}.`
        : ''}
    </p>
    <Link
      to="/"
      className="px-4 py-2 bg-sky-700 text-white rounded-md hover:bg-sky-800 transition-colors focus:outline-none focus:ring-2 focus:ring-sky-400 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
    >
      Back to home
    </Link>
  </div>
);

export default AccessDenied;
