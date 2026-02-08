import { test, expect } from '@playwright/test'

// E2E test for Spotify OAuth flow
test.describe('Spotify OAuth Authentication', () => {
    test('should display login button when not authenticated', async ({ page }) => {
        await page.goto('/')

        // Check that "Connecter Spotify" button is visible
        await expect(page.locator('button:has-text("Connecter Spotify")')).toBeVisible()
    })

    test('should display authentication status', async ({ page }) => {
        await page.goto('/')

        // Wait for auth status to load
        await page.waitForTimeout(500)

        // Should show either connected or not connected status
        const authStatus = page.locator('[data-testid="auth-status"]')
        await expect(authStatus).toBeVisible()
    })

    test('should redirect to Spotify when clicking login button', async ({ page, context }) => {
        await page.goto('/')

        // Click the "Connecter Spotify" button
        const loginButton = page.locator('button:has-text("Connecter Spotify")')
        await loginButton.click()

        await page.waitForURL(/\/auth\/spotify\/start|accounts\.spotify\.com/)
        expect(page.url()).toMatch(/\/auth\/spotify\/start|accounts\.spotify\.com/)
    })

    test('should display authenticated state when user is connected', async ({ page }) => {
        // Mock the auth status endpoint to return authenticated state
        await page.route('/api/auth/status', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    authenticated: true,
                    spotifyUserId: 'test-user-123',
                }),
            })
        })

        await page.goto('/')

        // Wait for auth status to load
        await page.waitForTimeout(500)

        // Should display connected status
        await expect(page.locator('text=Connecté')).toBeVisible()

        // Should display Spotify user ID
        await expect(page.locator('text=test-user-123')).toBeVisible()

        // Login button should not be visible when authenticated
        await expect(page.locator('button:has-text("Connecter Spotify")')).not.toBeVisible()
    })

    test('should display not authenticated state when user is not connected', async ({ page }) => {
        // Mock the auth status endpoint to return not authenticated state
        await page.route('/api/auth/status', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    authenticated: false,
                }),
            })
        })

        await page.goto('/')

        // Wait for auth status to load
        await page.waitForTimeout(500)

        // Should display not connected status
        await expect(page.locator('text=Non connecté')).toBeVisible()

        // Login button should be visible
        await expect(page.locator('button:has-text("Connecter Spotify")')).toBeVisible()
    })
})
