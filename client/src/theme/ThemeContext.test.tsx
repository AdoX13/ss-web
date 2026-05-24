import React from 'react';
import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { ThemeProvider, useTheme } from './ThemeContext';

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <ThemeProvider>{children}</ThemeProvider>
);

describe('ThemeContext', () => {
  it('toggles the theme, updates the <html> class, and persists it', () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    const initial = result.current.theme;

    act(() => result.current.toggleTheme());

    expect(result.current.theme).not.toBe(initial);
    expect(document.documentElement.classList.contains('dark')).toBe(
      result.current.theme === 'dark',
    );
    expect(localStorage.getItem('theme')).toBe(result.current.theme);
  });

  it('setTheme applies an explicit theme', () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    act(() => result.current.setTheme('dark'));
    expect(result.current.theme).toBe('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });
});
