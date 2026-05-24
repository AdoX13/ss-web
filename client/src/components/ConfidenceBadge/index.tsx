import React from 'react';

// Confidence pill. Mirrors the backend's 0.95 review threshold: green = high,
// amber = borderline, red = low/missing.
const ConfidenceBadge: React.FC<{ value: number }> = ({ value }) => {
  const pct = Math.round(value * 100);
  const tone =
    value >= 0.95
      ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
      : value >= 0.8
        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300'
        : 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300';
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${tone}`}
      title={`OCR confidence: ${pct}%`}
    >
      {pct}%
    </span>
  );
};

export default ConfidenceBadge;
