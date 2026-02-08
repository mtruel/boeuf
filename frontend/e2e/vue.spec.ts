import { test, expect } from '@playwright/test'

// E2E test validating frontend ↔ backend integration
test('should display app and connect to backend health endpoint', async ({ page }) => {
  await page.goto('/')

  // Check app title renders
  await expect(page.locator('h1')).toContainText('Boeuf')

  // Wait for backend health check to complete
  await page.waitForSelector('text=En ligne', { timeout: 5000 })

  // Verify backend status is displayed as online
  await expect(page.locator('text=En ligne')).toBeVisible()

  // Verify Button component renders
  await expect(page.locator('button:has-text("Actualiser")')).toBeVisible()
})

test('should handle health check refresh', async ({ page }) => {
  await page.goto('/')

  // Wait for initial health check
  await page.waitForSelector('text=En ligne')

  // Click refresh button
  await page.click('button:has-text("Actualiser")')

  // Verify status still shows online after refresh
  await expect(page.locator('text=En ligne')).toBeVisible()
})
