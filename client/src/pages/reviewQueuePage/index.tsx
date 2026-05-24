import React, { useCallback, useEffect, useState } from 'react';
import { useReviewQueue } from '../../hooks/useReviewQueue';
import { useReviewSocket } from '../../hooks/useReviewSocket';
import type { SocketStatus } from '../../hooks/useReviewSocket';
import type { ReviewItem, ReviewItemStatus } from '../../types/review';
import { REVIEW_STATUSES } from '../../types/review';
import {
  FIELD_LABELS,
  fieldLabel,
  isEnumField,
  ENUM_FIELD_OPTIONS,
} from '../../utils/fields';
import { stubReviewItems } from '../../utils/stubs';
import { timeAgo } from '../../utils/format';
import { ApiError } from '../../utils/api';
import ConfidenceBadge from '../../components/ConfidenceBadge';
import StatusBadge from '../../components/StatusBadge';
import {
  card,
  heading,
  label,
  input,
  errorBanner,
  subtleText,
} from '../../theme/styles';

const errMsg = (e: unknown, fallback: string) =>
  e instanceof ApiError ? e.message : fallback;

// Inline editor for correcting a field. Enum fields render a <select>.
const CorrectForm: React.FC<{
  item: ReviewItem;
  onSave: (value: string) => void;
  onCancel: () => void;
}> = ({ item, onSave, onCancel }) => {
  const [value, setValue] = useState(item.original_value ?? '');
  const enumField = isEnumField(item.field_name);

  return (
    <form
      className="mt-3 flex flex-wrap items-end gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (value.trim()) onSave(value.trim());
      }}
    >
      <div className="flex-1 min-w-[200px]">
        <label htmlFor={`correct-${item.id}`} className={label}>
          Corrected value
        </label>
        {enumField ? (
          <select
            id={`correct-${item.id}`}
            className={input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            autoFocus
          >
            <option value="">— select —</option>
            {ENUM_FIELD_OPTIONS[item.field_name].map((opt) => (
              <option key={opt} value={opt}>
                {opt}
              </option>
            ))}
          </select>
        ) : (
          <input
            id={`correct-${item.id}`}
            className={input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            autoFocus
          />
        )}
      </div>
      <button
        type="submit"
        disabled={!value.trim()}
        className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-indigo-400"
      >
        Save correction
      </button>
      <button
        type="button"
        onClick={onCancel}
        className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 rounded-md hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
      >
        Cancel
      </button>
    </form>
  );
};

const ReviewQueuePage: React.FC = () => {
  const [status, setStatus] = useState<ReviewItemStatus>('pending');
  const [fieldName, setFieldName] = useState('');
  const [polling, setPolling] = useState(true);
  const [sample, setSample] = useState<ReviewItem[] | null>(null);
  const [selected, setSelected] = useState(0);
  const [correctingId, setCorrectingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [socketStatus, setSocketStatus] = useState<SocketStatus>('closed');

  const { items, loading, error, refresh, approve, correct, reject } =
    useReviewQueue(
      { status, field_name: fieldName || undefined },
      polling && !sample ? 8000 : 0,
    );

  // Attempt live updates while viewing the pending queue; refresh on any push.
  useReviewSocket(
    status === 'pending' && !sample,
    () => refresh(),
    setSocketStatus,
  );

  const display = sample ?? items;

  useEffect(() => {
    // Keep selection in bounds when the list changes.
    setSelected((s) => Math.min(s, Math.max(0, display.length - 1)));
  }, [display.length]);

  const doApprove = useCallback(
    async (item: ReviewItem) => {
      setActionError(null);
      setCorrectingId(null);
      if (sample) {
        setSample((s) => s?.filter((i) => i.id !== item.id) ?? null);
        return;
      }
      try {
        await approve(item.id);
      } catch (e) {
        setActionError(errMsg(e, 'Failed to approve item.'));
      }
    },
    [sample, approve],
  );

  const doReject = useCallback(
    async (item: ReviewItem) => {
      setActionError(null);
      setCorrectingId(null);
      if (sample) {
        setSample((s) => s?.filter((i) => i.id !== item.id) ?? null);
        return;
      }
      try {
        await reject(item.id);
      } catch (e) {
        setActionError(errMsg(e, 'Failed to reject item.'));
      }
    },
    [sample, reject],
  );

  const doCorrect = useCallback(
    async (item: ReviewItem, value: string) => {
      setActionError(null);
      setCorrectingId(null);
      if (sample) {
        setSample((s) => s?.filter((i) => i.id !== item.id) ?? null);
        return;
      }
      try {
        await correct(item.id, value);
      } catch (e) {
        setActionError(errMsg(e, 'Failed to save correction.'));
      }
    },
    [sample, correct],
  );

  // Keyboard shortcuts for fast review (ignored while typing in a field).
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return;
      const current = display[selected];
      switch (e.key) {
        case 'j':
          setSelected((s) => Math.min(s + 1, display.length - 1));
          break;
        case 'k':
          setSelected((s) => Math.max(s - 1, 0));
          break;
        case 'a':
          if (current && current.status === 'pending') void doApprove(current);
          break;
        case 'r':
          if (current && current.status === 'pending') void doReject(current);
          break;
        case 'c':
          if (current && current.status === 'pending') setCorrectingId(current.id);
          break;
        case 'Escape':
          setCorrectingId(null);
          break;
        default:
          break;
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [display, selected, doApprove, doReject]);

  const live = socketStatus === 'open' ? 'live' : polling && !sample ? 'polling' : 'off';

  return (
    <div className="container mx-auto pb-10">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <h1 className={heading}>Review Queue</h1>
        <div className="flex items-center gap-3 text-sm">
          <span className="inline-flex items-center gap-1.5" title={`Updates: ${live}`}>
            <span
              className={`h-2.5 w-2.5 rounded-full ${
                live === 'off' ? 'bg-gray-400' : 'bg-green-500 animate-pulse'
              }`}
            />
            <span className={subtleText}>
              {live === 'live' ? 'Live' : live === 'polling' ? 'Auto-refresh' : 'Manual'}
            </span>
          </span>
          <button
            onClick={refresh}
            className="px-3 py-1.5 bg-sky-600 text-white rounded-md hover:bg-sky-700 transition-colors focus:outline-none focus:ring-2 focus:ring-sky-400"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className={`${card} p-4 mb-4 flex flex-wrap items-end gap-4`}>
        <div>
          <label htmlFor="status" className={label}>
            Status
          </label>
          <select
            id="status"
            className={input}
            value={status}
            onChange={(e) => setStatus(e.target.value as ReviewItemStatus)}
          >
            {REVIEW_STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="field" className={label}>
            Field
          </label>
          <select
            id="field"
            className={input}
            value={fieldName}
            onChange={(e) => setFieldName(e.target.value)}
          >
            <option value="">All fields</option>
            {Object.entries(FIELD_LABELS).map(([key, lbl]) => (
              <option key={key} value={key}>
                {lbl}
              </option>
            ))}
          </select>
        </div>
        <label className="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            type="checkbox"
            checked={polling}
            onChange={(e) => setPolling(e.target.checked)}
            className="rounded border-gray-300 text-sky-600 focus:ring-sky-500"
          />
          Auto-refresh (8s)
        </label>
        <span className={`ml-auto ${subtleText}`}>
          {display.length} item{display.length === 1 ? '' : 's'} ·{' '}
          <kbd className="px-1 rounded bg-gray-100 dark:bg-gray-700">j/k</kbd> move,{' '}
          <kbd className="px-1 rounded bg-gray-100 dark:bg-gray-700">a</kbd>pprove,{' '}
          <kbd className="px-1 rounded bg-gray-100 dark:bg-gray-700">r</kbd>eject,{' '}
          <kbd className="px-1 rounded bg-gray-100 dark:bg-gray-700">c</kbd>orrect
        </span>
      </div>

      {/* Audit hint */}
      <div className="mb-4 text-sm flex items-center gap-2 text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md px-3 py-2">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-4 w-4 flex-shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="10" />
          <path strokeLinecap="round" d="M12 16v-4m0-4h.01" />
        </svg>
        Every approve, correct, and reject action is recorded in the audit log.
      </div>

      {actionError && (
        <div className={`mb-4 ${errorBanner}`} role="alert">
          {actionError}
        </div>
      )}

      {/* List */}
      {loading ? (
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500" />
        </div>
      ) : display.length > 0 ? (
        <>
          {error && !sample && (
            <div className={`mb-4 ${errorBanner}`} role="alert">
              {error}
            </div>
          )}
          <ul className="space-y-3">
          {display.map((item, idx) => {
            const isSelected = idx === selected;
            const isPending = item.status === 'pending';
            return (
              <li
                key={item.id}
                onClick={() => setSelected(idx)}
                className={`${card} p-4 cursor-pointer transition-shadow ${
                  isSelected ? 'ring-2 ring-sky-400' : 'hover:shadow-lg'
                }`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium text-gray-900 dark:text-gray-100">
                        {fieldLabel(item.field_name)}
                      </span>
                      <ConfidenceBadge value={item.original_confidence} />
                      <StatusBadge status={item.status} />
                    </div>
                    <p className="mt-1 text-sm text-gray-700 dark:text-gray-300 break-words">
                      OCR value:{' '}
                      {item.original_value ? (
                        <span className="font-mono">{item.original_value}</span>
                      ) : (
                        <span className="italic text-gray-400">empty</span>
                      )}
                    </p>
                    <p className="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
                      Image {item.image_id} · {timeAgo(item.created_at)}
                      {item.reviewer_email ? ` · by ${item.reviewer_email}` : ''}
                      {item.corrected_value
                        ? ` · → ${item.corrected_value}`
                        : ''}
                    </p>
                  </div>

                  {isPending && (
                    <div className="flex gap-2 flex-shrink-0">
                      <button
                        onClick={() => void doApprove(item)}
                        className="px-3 py-1.5 text-sm bg-green-600 text-white rounded-md hover:bg-green-700 transition-colors focus:outline-none focus:ring-2 focus:ring-green-400"
                      >
                        Approve
                      </button>
                      <button
                        onClick={() =>
                          setCorrectingId((id) =>
                            id === item.id ? null : item.id,
                          )
                        }
                        className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-400"
                      >
                        Correct
                      </button>
                      <button
                        onClick={() => void doReject(item)}
                        className="px-3 py-1.5 text-sm bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors focus:outline-none focus:ring-2 focus:ring-red-400"
                      >
                        Reject
                      </button>
                    </div>
                  )}
                </div>

                {correctingId === item.id && (
                  <CorrectForm
                    item={item}
                    onSave={(value) => void doCorrect(item, value)}
                    onCancel={() => setCorrectingId(null)}
                  />
                )}
              </li>
            );
          })}
          </ul>
        </>
      ) : error ? (
        <div className={errorBanner} role="alert">
          <p className="font-medium">Could not load the review queue</p>
          <p className="mt-1">{error}</p>
          {import.meta.env.DEV && (
            <button
              onClick={() => setSample(stubReviewItems)}
              className="mt-3 px-3 py-1.5 text-sm bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 rounded-md hover:bg-gray-300 dark:hover:bg-gray-600"
            >
              Load sample data (dev)
            </button>
          )}
        </div>
      ) : (
        <div className={`${card} text-center py-12 ${subtleText}`}>
          Nothing to review. The queue is clear for “{status}”.
        </div>
      )}
    </div>
  );
};

export default ReviewQueuePage;
