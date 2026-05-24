import React from 'react';
import type { ReviewItemStatus } from '../../types/review';

const TONES: Record<ReviewItemStatus, string> = {
  pending:
    'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300',
  approved:
    'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300',
  corrected:
    'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-300',
  rejected:
    'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300',
};

const StatusBadge: React.FC<{ status: ReviewItemStatus }> = ({ status }) => (
  <span
    className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium capitalize ${TONES[status]}`}
  >
    {status}
  </span>
);

export default StatusBadge;
