import { test, expect } from '@playwright/test'

async function mockHealth(page: any) {
  await page.route('**/api/health', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok' })
    })
  })
}

test.describe('Story 1.9 auth and device flows', () => {
  test('redirects unauthenticated session route and shows toast (AC 4)', async ({ page }) => {
    await mockHealth(page)
    await page.route('**/api/auth/status', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authenticated: false })
      })
    })

    await page.goto('/session/test-session')
    await expect(page).toHaveURL(/\?redirect=\/session\/test-session/)
    await expect(page.getByText('Please sign in to access this session')).toBeVisible()
  })

  test('redirects unauthenticated join route and preserves invite (AC 5)', async ({ page }) => {
    await mockHealth(page)
    await page.route('**/api/auth/status', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authenticated: false })
      })
    })

    await page.goto('/join/INVITE_CODE')
    await expect(page).toHaveURL(/\?invite=INVITE_CODE/)
    await expect(page.getByText('Sign in to join this session')).toBeVisible()
  })

  test('logout hides session section immediately (AC 3)', async ({ page }) => {
    await mockHealth(page)

    let loggedOut = false
    await page.route('**/api/auth/status', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          authenticated: !loggedOut,
          spotifyUserId: loggedOut ? undefined : 'test-user-123'
        })
      })
    })

    await page.route('**/api/auth/logout', async (route: any) => {
      loggedOut = true
      await route.fulfill({ status: 200 })
    })

    await page.goto('/')

    await expect(page.getByText('Connecté')).toBeVisible()
    await expect(page.getByTestId('create-session-btn')).toBeVisible()

    await page.getByTestId('logout-button').click()

    await expect(page.getByText('Non connecté')).toBeVisible()
    await expect(page.getByTestId('create-session-btn')).toBeHidden()
  })

  test('shows device error toast on sync start (AC 2)', async ({ page }) => {
    await page.route('**/api/auth/status', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authenticated: true, spotifyUserId: 'test-user-123' })
      })
    })

    await page.route('**/api/sessions/**', async (route: any) => {
      const url = new URL(route.request().url())
      if (url.pathname.endsWith('/sync/start')) {
        await route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 'SPOTIFY_NO_DEVICE',
            message: 'No active Spotify device found.',
            requiresActiveDevice: true,
            suggestedAction: 'OPEN_SPOTIFY_WEB_PLAYER'
          })
        })
        return
      }

      if (url.pathname.endsWith('/me')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ syncState: 'ready' })
        })
        return
      }

      if (url.pathname.endsWith('/player/state')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            state: {
              isPlaying: false,
              positionMs: 0,
              track: null
            }
          })
        })
        return
      }

      if (url.pathname.split('/').length === 4) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            sessionId: 'sess_test',
            nowPlaying: null
          })
        })
        return
      }

      await route.fulfill({ status: 404 })
    })

    await page.goto('/session/sess_test')
    await expect(page.getByRole('button', { name: "Démarrer l'écoute" })).toBeVisible()

    await page.getByRole('button', { name: "Démarrer l'écoute" }).click()
    await expect(page.getByText('No Active Spotify Device')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Open Spotify' })).toBeVisible()
  })

  test('retry backoff delays sync restart (AC 8)', async ({ page }) => {
    const syncStartTimings: number[] = []

    await page.route('**/api/auth/status', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authenticated: true, spotifyUserId: 'test-user-123' })
      })
    })

    await page.route('**/api/sessions/**', async (route: any) => {
      const url = new URL(route.request().url())
      if (url.pathname.endsWith('/sync/start')) {
        syncStartTimings.push(Date.now())
        const callCount = syncStartTimings.length

        if (callCount === 1) {
          await route.fulfill({
            status: 503,
            contentType: 'application/json',
            body: JSON.stringify({
              code: 'SPOTIFY_UNAVAILABLE',
              message: 'Spotify temporarily unavailable.'
            })
          })
          return
        }

        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            nowPlaying: {
              trackId: 'track_1',
              trackName: 'Test Track',
              artist: 'Test Artist',
              isPlaying: true,
              positionMs: 0,
              durationMs: 180000
            }
          })
        })
        return
      }

      if (url.pathname.endsWith('/me')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ syncState: 'ready' })
        })
        return
      }

      if (url.pathname.endsWith('/player/state')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            state: {
              isPlaying: false,
              positionMs: 0,
              track: null
            }
          })
        })
        return
      }

      if (url.pathname.split('/').length === 4) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            sessionId: 'sess_test',
            nowPlaying: null
          })
        })
        return
      }

      await route.fulfill({ status: 404 })
    })

    await page.goto('/session/sess_test')
    await expect(page.getByRole('button', { name: "Démarrer l'écoute" })).toBeVisible()

    await page.getByRole('button', { name: "Démarrer l'écoute" }).click()
    await expect(page.getByRole('button', { name: 'Réessayer' })).toBeVisible()

    await page.getByRole('button', { name: 'Réessayer' }).click()

    await expect.poll(() => syncStartTimings.length, {
      timeout: 6000
    }).toBe(2)

    expect(syncStartTimings[1] - syncStartTimings[0]).toBeGreaterThanOrEqual(1900)
    await expect(page.getByText("Synced! You're now listening together.")).toBeVisible()
  })
})
