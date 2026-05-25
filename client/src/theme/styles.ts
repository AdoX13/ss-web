// Shared Tailwind class strings. Centralising these keeps the look consistent
// and — crucially — keeps every form control and surface dark-mode aware in one
// place rather than scattering `dark:` variants across dozens of files.

export const card =
  'bg-white dark:bg-gray-800 rounded-lg shadow-md border border-transparent dark:border-gray-700';

export const cardSubtle =
  'bg-gray-50 dark:bg-gray-800/50 rounded-lg shadow-sm border border-transparent dark:border-gray-700';

export const label =
  'block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1';

export const input =
  'w-full px-3 py-2 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-sky-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed';

export const heading = 'text-2xl font-semibold text-sky-700 dark:text-sky-300';

export const subtleText = 'text-gray-600 dark:text-gray-400';

export const errorBanner =
  'p-3 rounded-md border bg-red-50 border-red-300 text-red-700 dark:bg-red-900/30 dark:border-red-800 dark:text-red-300';

export const successBanner =
  'p-3 rounded-md border bg-green-50 border-green-300 text-green-700 dark:bg-green-900/30 dark:border-green-800 dark:text-green-300';

export const tableHeadCell =
  'px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400';

export const tableCell =
  'px-4 py-2 text-sm text-gray-800 dark:text-gray-200 border-t border-gray-100 dark:border-gray-700';
