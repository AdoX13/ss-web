import { test, expect } from '@playwright/test';
import { loginAs } from './utils';

test('researcher lists and runs a report', async ({ page }) => {
  await page.route(/\/api\/v1\/reports(\?|$)/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { name: 'recent_exams', description: 'Recent exams', roles: [] },
      ]),
    }),
  );
  // Run endpoint returns the CAPITALISED shape the Go backend currently emits.
  await page.route(/\/api\/v1\/reports\/recent_exams/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        Name: 'recent_exams',
        Columns: ['month', 'count'],
        Rows: [
          { month: 'Jan', count: 3 },
          { month: 'Feb', count: 5 },
        ],
      }),
    }),
  );

  await loginAs(page, 'researcher');
  await page.goto('/reports');

  await page.getByRole('link', { name: /Recent exams/i }).click();
  await expect(page).toHaveURL(/\/reports\/recent_exams$/);

  // Table renders despite the capitalised API keys (normaliser handles it).
  await expect(page.getByRole('cell', { name: 'Jan' })).toBeVisible();
  await expect(page.getByRole('cell', { name: '5' })).toBeVisible();
});
