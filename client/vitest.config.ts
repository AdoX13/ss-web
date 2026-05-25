import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Separate from vite.config.ts so the production build config stays clean.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    css: false,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Unit-coverage scope = the logic layer + reusable components. Page
      // components and presentational chrome (navbar, buttons, cards, theme
      // toggle) are exercised by the Playwright E2E suite (see e2e/) across all
      // four roles, which is the right tool for full-page behaviour.
      include: [
        'src/utils/api.ts',
        'src/utils/fields.ts',
        'src/utils/format.ts',
        'src/hooks/**',
        'src/contexts/**',
        'src/theme/ThemeContext.tsx',
        'src/components/ProtectedRoute.tsx',
        'src/components/AccessDenied/**',
        'src/components/ConfidenceBadge/**',
        'src/components/StatusBadge/**',
        'src/types/auth.ts',
        'src/types/review.ts',
      ],
      thresholds: { lines: 75, statements: 75, functions: 75, branches: 75 },
    },
  },
});
