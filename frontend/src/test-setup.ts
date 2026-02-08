import { vi, beforeEach } from 'vitest'
import { config } from '@vue/test-utils'
import { RouteLocationNormalizedLoaded } from 'vue-router'

// Mock fetch globally for tests to avoid "Failed to parse URL" errors
globalThis.fetch = vi.fn() as any

// Mock Vue Router globally to avoid injection warnings in tests
// Provide both mocks and stubs for comprehensive coverage
config.global.mocks = {
    $route: {
        params: {},
        query: {},
        path: '/',
        name: undefined,
        meta: {},
        fullPath: '/',
        hash: '',
        matched: []
    } as RouteLocationNormalizedLoaded,
    $router: {
        push: vi.fn(),
        replace: vi.fn(),
        go: vi.fn(),
        back: vi.fn(),
        forward: vi.fn(),
        resolve: vi.fn(),
        currentRoute: {
            value: {
                params: {},
                query: {},
                path: '/'
            }
        }
    }
}

// Stub RouterLink and RouterView to avoid router dependency issues
config.global.stubs = {
    RouterLink: true,
    RouterView: true
}

// Reset mocks before each test
beforeEach(() => {
    vi.clearAllMocks()

        // Default mock implementation that returns ok health response
        ; (globalThis.fetch as any).mockResolvedValue({
            ok: true,
            json: async () => ({ status: 'ok' }),
        })
})
