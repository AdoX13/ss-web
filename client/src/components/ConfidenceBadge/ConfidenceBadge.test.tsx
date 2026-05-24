import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ConfidenceBadge from './index';

describe('ConfidenceBadge', () => {
  it('renders the confidence as a rounded percentage', () => {
    render(<ConfidenceBadge value={0.714} />);
    expect(screen.getByText('71%')).toBeInTheDocument();
  });

  it('renders the 0% and 100% bounds', () => {
    const { rerender } = render(<ConfidenceBadge value={0} />);
    expect(screen.getByText('0%')).toBeInTheDocument();
    rerender(<ConfidenceBadge value={1} />);
    expect(screen.getByText('100%')).toBeInTheDocument();
  });

  it('exposes the confidence in the title for screen readers', () => {
    render(<ConfidenceBadge value={0.5} />);
    expect(screen.getByTitle('OCR confidence: 50%')).toBeInTheDocument();
  });
});
