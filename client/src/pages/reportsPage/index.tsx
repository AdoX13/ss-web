import React from 'react';
import { Link } from 'react-router-dom';
import { useReportList } from '../../hooks/useReports';
import { ROLE_LABELS } from '../../types/auth';
import { card, heading, subtleText, errorBanner } from '../../theme/styles';

// Landing page: lists the reports the current role is allowed to run (the
// backend filters the list by role).
const ReportsPage: React.FC = () => {
  const { reports, loading, error } = useReportList();

  return (
    <div className="container mx-auto pb-10">
      <h1 className={`${heading} mb-2`}>Reports</h1>
      <p className={`${subtleText} mb-6`}>
        Compliance and operational reports. Select one to run it over a date
        range and export the results.
      </p>

      {loading ? (
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500" />
        </div>
      ) : error ? (
        <div className={errorBanner} role="alert">
          {error}
        </div>
      ) : reports.length === 0 ? (
        <div className={`${card} text-center py-12 ${subtleText}`}>
          No reports are available for your role.
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {reports.map((r) => (
            <Link
              key={r.name}
              to={`/reports/${r.name}`}
              className={`${card} p-5 hover:shadow-lg transition-shadow focus:outline-none focus:ring-2 focus:ring-sky-400`}
            >
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
                {r.description || r.name}
              </h2>
              <p className="text-xs font-mono text-gray-500 dark:text-gray-400 mb-3">
                {r.name}
              </p>
              {r.roles && r.roles.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {r.roles.map((role) => (
                    <span
                      key={role}
                      className="px-2 py-0.5 rounded-full text-xs bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300"
                    >
                      {ROLE_LABELS[role] ?? role}
                    </span>
                  ))}
                </div>
              ) : (
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  All roles
                </span>
              )}
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export default ReportsPage;
