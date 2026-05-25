import { test, expect } from '@playwright/test';
import { loginAs } from './utils';
import type { Role } from './utils';

const NAV = [
  'Photos',
  'Devices',
  'Statistics',
  'Reports',
  'Review Queue',
  'Admin',
] as const;

// Which nav items each role should see (mirrors App.tsx route guards).
const VISIBLE: Record<Role, string[]> = {
  admin: ['Photos', 'Devices', 'Statistics', 'Reports', 'Review Queue', 'Admin'],
  doctor: ['Photos', 'Devices', 'Statistics', 'Reports', 'Review Queue'],
  researcher: ['Statistics', 'Reports'],
  auditor: ['Reports'],
};

for (const role of Object.keys(VISIBLE) as Role[]) {
  test(`${role} sees only its permitted nav items`, async ({ page }) => {
    await loginAs(page, role);
    for (const item of NAV) {
      const locator = page.getByRole('button', { name: item, exact: true });
      if (VISIBLE[role].includes(item)) {
        await expect(locator, `${role} should see ${item}`).toBeVisible();
      } else {
        await expect(locator, `${role} should NOT see ${item}`).toHaveCount(0);
      }
    }
  });
}

test('auditor is blocked from the review queue (RBAC guard)', async ({ page }) => {
  await loginAs(page, 'auditor');
  await page.goto('/review-queue');
  await expect(
    page.getByRole('heading', { name: 'Access denied' }),
  ).toBeVisible();
});

test('researcher is blocked from admin (RBAC guard)', async ({ page }) => {
  await loginAs(page, 'researcher');
  await page.goto('/admin/users');
  await expect(
    page.getByRole('heading', { name: 'Access denied' }),
  ).toBeVisible();
});

test('logged-out users are redirected to login', async ({ page }) => {
  await page.goto('/review-queue');
  await expect(page).toHaveURL(/\/login$/);
});
