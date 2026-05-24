import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusBadge from './index';
import { REVIEW_STATUSES } from '../../types/review';

describe('StatusBadge', () => {
  it('renders every review status label', () => {
    for (const status of REVIEW_STATUSES) {
      const { unmount } = render(<StatusBadge status={status} />);
      expect(screen.getByText(status)).toBeInTheDocument();
      unmount();
    }
  });
});
