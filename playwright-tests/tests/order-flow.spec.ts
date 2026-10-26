import { test, expect, Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

/**
 * Roles supported by the delivery application.
 */
enum UserRole {
  Customer = 'customer',
  Restaurant = 'restaurant',
  Rider = 'rider',
}

/**
 * Authenticates a user with email and password via the login form.
 *
 * @param page - Playwright page instance
 * @param email - User email address
 * @param password - User password
 *
 * @example
 * await login(page, 'customer1@example.com', 'password123');
 */
async function login(page: Page, email: string, password: string): Promise<void> {
  await page.goto('/login');
  await page.locator('input[name="email"]').fill(email);
  await page.locator('input[name="password"]').fill(password);
  await page.getByRole('button', { name: 'Log In' }).click();
}

/**
 * Performs user logout via the logout form submission and asserts redirection to login.
 *
 * @param page - Playwright page instance
 *
 * @example
 * await logout(page);
 */
async function logout(page: Page): Promise<void> {
  await page.waitForLoadState('load');
  const logoutForm = page.locator('form[action="/logout"]');
  if ((await logoutForm.count()) > 0) {
    await Promise.all([
      page.waitForURL(/\/login/),
      logoutForm.evaluate((form: HTMLFormElement) => form.submit()),
    ]);
  } else {
    await page.goto('/login');
  }
  await expect(page).toHaveURL(/\/login/);
}

test.describe.serial('Food Delivery End-to-End Order Lifecycle', () => {
  const rootDemo1 = path.resolve(__dirname, '..', '..', 'demo1.png');
  const rootDemo2 = path.resolve(__dirname, '..', '..', 'demo2.png');
  const demo1Path = fs.existsSync(rootDemo1) ? rootDemo1 : path.resolve(__dirname, '..', 'demo1.png');
  const demo2Path = fs.existsSync(rootDemo2) ? rootDemo2 : path.resolve(__dirname, '..', 'demo2.png');

  test('complete order flow: customer checkout -> restaurant preparation -> rider delivery with proofs', async ({ page }) => {
    // -------------------------------------------------------------------------
    // Step 1: Customer Login, Add Menu Item to Cart, and Checkout
    // -------------------------------------------------------------------------
    await login(page, 'customer1@example.com', 'password123');
    await expect(page).toHaveURL('/');

    // Add first available menu item to cart
    const addToCartButton = page.getByRole('button', { name: 'Add to Cart' }).first();
    await expect(addToCartButton).toBeVisible();
    await addToCartButton.click();

    // Navigate to Cart
    await page.locator('#nav-cart-btn').click();
    await expect(page).toHaveURL('/cart');
    await expect(page.getByText('Your Basket & Checkout')).toBeVisible();

    // Place Order (Cash on Delivery)
    const placeOrderButton = page.getByRole('button', { name: 'Place Order (Cash on Delivery)' });
    await expect(placeOrderButton).toBeVisible();
    await placeOrderButton.click();

    // Assert order placed and extract generated Order ID
    await expect(page).toHaveURL(/\/orders\/\d+/);
    const urlMatch = page.url().match(/\/orders\/(\d+)/);
    expect(urlMatch).not.toBeNull();
    const orderId = urlMatch![1];

    await expect(page.getByText(`Order #${orderId}`)).toBeVisible();
    await expect(page.getByText('Pending Confirmation')).toBeVisible();

    await logout(page);

    // -------------------------------------------------------------------------
    // Step 2: Restaurant Login, Go to Orders, Mark Preparing, and Mark Ready
    // -------------------------------------------------------------------------
    await login(page, 'owner1@example.com', 'password123');
    await expect(page).toHaveURL(/\/restaurant/);

    // Go to restaurant orders
    await page.getByRole('link', { name: 'Restaurant Orders' }).click();
    await expect(page).toHaveURL(/\/restaurant\/orders/);

    // Locate the row for the newly created order
    const orderRow = page.locator('tbody tr', { hasText: `#${orderId}` });
    await expect(orderRow).toBeVisible();
    await expect(orderRow.getByText('Pending')).toBeVisible();

    // Click preparing (Confirm button sets status to confirmed)
    const confirmButton = orderRow.getByRole('button', { name: 'Confirm' });
    await expect(confirmButton).toBeVisible();
    await Promise.all([
      page.waitForNavigation(),
      confirmButton.click(),
    ]);

    await expect(orderRow.getByText('Confirmed (Preparing)')).toBeVisible();

    // Click ready for pick up
    const readyButton = orderRow.getByRole('button', { name: 'Ready for Pickup' });
    await expect(readyButton).toBeVisible();
    await Promise.all([
      page.waitForNavigation(),
      readyButton.click(),
    ]);

    await expect(orderRow.getByText('Ready for Pickup')).toBeVisible();

    await logout(page);

    // -------------------------------------------------------------------------
    // Step 3: Rider Login, Accept Order, Upload Proof 1 (Pickup) & Proof 2 (Delivery)
    // -------------------------------------------------------------------------
    await login(page, 'rider1@example.com', 'password123');
    await expect(page).toHaveURL('/rider');

    // Accept order from available orders table
    const availableOrderRow = page.locator('tbody tr', { hasText: `#${orderId}` });
    await expect(availableOrderRow).toBeVisible();
    await availableOrderRow.getByRole('button', { name: 'Accept Order' }).click();

    // Order is now active; upload demo1.png for pickup proof
    const pickupForm = page.locator(`form[action="/rider/orders/${orderId}/start"]`);
    await expect(pickupForm).toBeVisible();
    await pickupForm.locator('input[name="proof_image"]').setInputFiles(demo1Path);
    await pickupForm.getByRole('button', { name: 'Upload Proof & Start Delivery' }).click();

    // Verify pickup proof uploaded and status changed to In Delivery
    await expect(page.getByText('Pickup Proof Uploaded')).toBeVisible();
    await expect(page.locator('.badge:has-text("In Delivery")')).toBeVisible();

    // Upload demo2.png for delivery proof
    const deliveryForm = page.locator(`form[action="/rider/orders/${orderId}/complete"]`);
    await expect(deliveryForm).toBeVisible();
    await deliveryForm.locator('input[name="proof_image"]').setInputFiles(demo2Path);
    await deliveryForm.getByRole('button', { name: 'Upload Proof & Complete Delivery' }).click();

    // Verify delivery completed and displayed in history
    await expect(page.getByText('Recent Completed Deliveries')).toBeVisible();
    await expect(page.locator(`.list-group-item:has-text("Order #${orderId}")`)).toBeVisible();
  });
});
