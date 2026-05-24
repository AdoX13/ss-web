import { test, expect } from '@playwright/test';
import { loginAs } from './utils';

const items = [
  {
    id: 'a1',
    image_id: 'img1',
    field_name: 'patient_name',
    original_value: 'Popescu Ion',
    original_confidence: 0.71,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
  {
    id: 'a2',
    image_id: 'img1',
    field_name: 'control_type',
    original_value: null,
    original_confidence: 0,
    status: 'pending',
    created_at: new Date().toISOString(),
  },
];

test('doctor reviews the queue and approves an item', async ({ page }) => {
  // List endpoint (with or without query string), excluding action sub-paths.
  await page.route(/\/api\/v1\/review-queue(\?|$)/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(items),
    }),
  );
  await page.route(/\/review-queue\/[^/]+\/approve$/, (route) =>
    route.fulfill({ status: 204 }),
  );

  await loginAs(page, 'doctor');
  await page.goto('/review-queue');

  // Field labels are mapped to Romanian and the OCR value is shown.
  // Scope to the list so we don't match the field-filter <option>.
  await expect(page.getByRole('list').getByText('Nume pacient')).toBeVisible();
  await expect(page.getByText('Popescu Ion')).toBeVisible();
  await expect(page.getByText('71%')).toBeVisible();

  // Approving fires the POST to the approve endpoint.
  const approveReq = page.waitForRequest(/\/review-queue\/[^/]+\/approve$/);
  await page.getByRole('button', { name: 'Approve' }).first().click();
  await approveReq;
});

test('correcting an enum field shows a select, not a text box', async ({
  page,
}) => {
  await page.route(/\/api\/v1\/review-queue(\?|$)/, (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(items),
    }),
  );

  await loginAs(page, 'doctor');
  await page.goto('/review-queue');

  // The control_type row (enum field) → corrector renders a <select>.
  await page
    .locator('li', { hasText: 'Tip control' })
    .getByRole('button', { name: 'Correct' })
    .click();
  const select = page.getByLabel('Corrected value');
  await expect(select).toBeVisible();
  await expect(select.locator('option', { hasText: 'Periodic' })).toHaveCount(1);
});
