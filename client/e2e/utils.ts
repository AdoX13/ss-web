import { expect } from '@playwright/test';
import type { Page } from '@playwright/test';

export type Role = 'admin' | 'doctor' | 'researcher' | 'auditor';

// Mock the login endpoint to return a session for the given role.
export async function mockLogin(page: Page, role: Role) {
  await page.route('**/api/v1/auth/login', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        access_token: 'e2e.access.token',
        refresh_token: 'e2e.refresh.token',
        token_type: 'Bearer',
        email: `${role}@firstforce.local`,
        role,
      }),
    }),
  );
}

// Drive the real login form and assert we end up authenticated.
export async function loginAs(page: Page, role: Role) {
  await mockLogin(page, role);
  await page.goto('/login');
  await page.getByLabel('Email Address').fill(`${role}@firstforce.local`);
  await page.getByLabel('Password').fill('e2e-password');
  // Scope to <main> so we don't match the navbar's "Login" button.
  await page.getByRole('main').getByRole('button', { name: 'Login' }).click();
  await expect(page.getByRole('button', { name: 'Logout' })).toBeVisible();
}
