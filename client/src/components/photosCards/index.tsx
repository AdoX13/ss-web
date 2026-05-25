import React, { useState, useEffect } from 'react';
import fallbackImage from '../../assets/photo-fallback.svg';
import { apiFetch } from '../../utils/api';

interface PhotoCardProps {
  photoId: string;
  imageUrl: string;
  altText?: string;
  extractedText?: string;
  isAdmin?: boolean;
  needsReview?: boolean;
  onDelete?: (photoId: string) => void;
}

const PhotoCard: React.FC<PhotoCardProps> = ({
  photoId,
  imageUrl,
  altText = 'Photo',
  extractedText = '',
  isAdmin = false,
  needsReview = false,
  onDelete,
}) => {
  const [isZoomed, setIsZoomed] = useState(false);
  const [resolvedSrc, setResolvedSrc] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const toggleZoom = () => setIsZoomed(!isZoomed);

  // The /uploads route is auth-gated, so a plain <img src> would 401. Fetch the
  // image with the bearer token and serve it as an object URL.
  useEffect(() => {
    if (!imageUrl) {
      setResolvedSrc(null);
      return;
    }
    let active = true;
    let objectUrl: string | null = null;
    apiFetch(imageUrl)
      .then((res) =>
        res.ok ? res.blob() : Promise.reject(new Error(String(res.status))),
      )
      .then((blob) => {
        if (!active) return;
        objectUrl = URL.createObjectURL(blob);
        setResolvedSrc(objectUrl);
      })
      .catch(() => {
        if (active) setResolvedSrc(null);
      });
    return () => {
      active = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [imageUrl]);

  // Close the zoom modal with Escape.
  useEffect(() => {
    if (!isZoomed) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setIsZoomed(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [isZoomed]);

  const handleModalClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) setIsZoomed(false);
  };

  const handleConfirmDelete = async () => {
    setIsDeleting(true);
    if (onDelete) await onDelete(photoId);
    setShowDeleteConfirm(false);
    setIsDeleting(false);
  };

  return (
    <>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-transparent dark:border-gray-700 overflow-hidden transition-all hover:shadow-lg relative">
        <div className="relative h-48 cursor-pointer" onClick={toggleZoom}>
          <img
            src={resolvedSrc ?? fallbackImage}
            alt={altText}
            onError={() => setResolvedSrc(null)}
            className="w-full h-full object-cover"
          />
          {needsReview && (
            <span className="absolute top-2 left-2 bg-yellow-400 text-yellow-900 text-xs font-bold px-2 py-0.5 rounded shadow">
              Review needed
            </span>
          )}
          {isAdmin && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowDeleteConfirm(true);
              }}
              className="absolute top-2 right-2 bg-red-500 hover:bg-red-600 text-white rounded-full p-2 shadow-lg transition-all duration-200 opacity-80 hover:opacity-100"
              title="Delete photo"
              aria-label="Delete photo"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          )}
        </div>

        {extractedText && (
          <div className="p-3 border-t border-gray-100 dark:border-gray-700">
            <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
              {extractedText}
            </p>
          </div>
        )}

        {showDeleteConfirm && (
          <div
            className="absolute inset-0 bg-black/50 flex items-center justify-center"
            role="dialog"
            aria-modal="true"
            aria-label="Confirm delete photo"
          >
            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 m-4 shadow-xl">
              <p className="text-gray-800 dark:text-gray-200 mb-4">
                Delete this photo?
              </p>
              <div className="flex gap-2 justify-center">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-4 py-2 bg-gray-300 dark:bg-gray-600 dark:text-gray-100 hover:bg-gray-400 dark:hover:bg-gray-500 rounded-md transition-colors"
                  disabled={isDeleting}
                >
                  Cancel
                </button>
                <button
                  onClick={handleConfirmDelete}
                  className="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-md transition-colors"
                  disabled={isDeleting}
                >
                  {isDeleting ? 'Deleting…' : 'Delete'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {isZoomed && (
        <div
          className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 transition-opacity duration-300 ease-in-out"
          onClick={handleModalClick}
          role="dialog"
          aria-modal="true"
          aria-label={altText}
        >
          <div className="relative bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-4xl max-h-[90vh] overflow-hidden transform transition-all duration-300 ease-in-out animate-scaleIn">
            <div className="absolute top-0 right-0 left-0 bg-gradient-to-b from-black/50 to-transparent h-20 z-10 flex justify-between items-start p-4">
              <div className="text-white text-lg font-medium truncate pr-10">
                {altText}
              </div>
              <button
                className="bg-white/20 hover:bg-white/40 text-white rounded-full p-2 backdrop-blur-sm transition-all duration-200"
                onClick={toggleZoom}
                aria-label="Close"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            <div className="p-4 pt-20">
              <img
                src={resolvedSrc ?? fallbackImage}
                alt={altText}
                className="max-w-full max-h-[65vh] object-contain mx-auto rounded-md"
              />
            </div>

            {extractedText && (
              <div className="bg-gray-50 dark:bg-gray-700/50 p-6 border-t border-gray-100 dark:border-gray-700">
                <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">
                  Extracted Text
                </h3>
                <p className="text-gray-800 dark:text-gray-200 text-base">
                  {extractedText}
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
};

export default PhotoCard;
