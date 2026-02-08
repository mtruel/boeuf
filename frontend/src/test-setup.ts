import { vi, beforeEach } from 'vitest'

// Mock fetch globally for tests to avoid "Failed to parse URL" errors
globalThis.fetch = vi.fn() as any

// Reset mocks before each test
beforeEach(() => {
    vi.clearAllMocks()

        // Default mock implementation that returns ok health response
        ; (globalThis.fetch as any).mockResolvedValue({
            ok: true,
            json: async () => ({ status: 'ok' }),
        })
})
