import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { loginAs } from './utils';
import type { Page } from '@playwright/test';

// Return only serious/critical WCAG 2 A/AA violations, simplified for readable
// failure output.
async function seriousViolations(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  return results.violations
    .filter((v) => v.impact === 'serious' || v.impact === 'critical')
    .map((v) => ({ id: v.id, impact: v.impact, help: v.help }));
}

test('home page has no serious a11y violations', async ({ page }) => {
  await page.goto('/');
  expect(await seriousViolations(page)).toEqual([]);
});

test('login page has no serious a11y violations', async ({ page }) => {
  await page.goto('/login');
  expect(await seriousViolations(page)).toEqual([]);
});

test('review queue has no serious a11y violations', async ({ page }) => {
  await page.route(/\/api\/v1\/review-queue(\?|$)/, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: '[]' }),
  );
  await loginAs(page, 'admin');
  await page.goto('/review-queue');
  await expect(page.getByText(/Nothing to review/)).toBeVisible();
  expect(await seriousViolations(page)).toEqual([]);
});

test('reports landing has no serious a11y violations', async ({ page }) => {
  await page.route(/\/api\/v1\/reports(\?|$)/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { name: 'recent_exams', description: 'Recent exams', roles: [] },
      ]),
    }),
  );
  await loginAs(page, 'researcher');
  await page.goto('/reports');
  await expect(page.getByRole('link', { name: /Recent exams/i })).toBeVisible();
  expect(await seriousViolations(page)).toEqual([]);
});
