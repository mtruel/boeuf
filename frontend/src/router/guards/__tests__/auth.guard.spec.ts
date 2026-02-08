import { describe, it, expect, vi, beforeEach } from 'vitest'
import { authGuard, resetAuthCache } from '../auth.guard'
import type { RouteLocationNormalized } from 'vue-router'

// Mock dependencies
vi.mock('@/composables/useToast', () => ({
    useToast: () => ({
        warning: vi.fn()
    })
}))

global.fetch = vi.fn()

describe('authGuard (AC 4, 5)', () => {
    let next: ReturnType<typeof vi.fn>

    beforeEach(() => {
        vi.clearAllMocks()
        resetAuthCache() // Clear auth cache between tests
        next = vi.fn()
    })

    const createMockRoute = (path: string, params: Record<string, any> = {}): RouteLocationNormalized => ({
        path,
        params,
        query: {},
        hash: '',
        fullPath: path,
        name: undefined,
        meta: {},
        matched: [],
        redirectedFrom: undefined
    })

    it('should allow access to non-protected routes', async () => {
        const to = createMockRoute('/')
        const from = createMockRoute('/')

        await authGuard(to, from, next)

        expect(next).toHaveBeenCalledWith()
        expect(fetch).not.toHaveBeenCalled()
    })

    it('should allow authenticated user to access /session route (AC 4)', async () => {
        const to = createMockRoute('/session/abc123')
        const from = createMockRoute('/')

        ; (fetch as any).mockResolvedValueOnce({
            json: async () => ({ authenticated: true })
        })

        await authGuard(to, from, next)

        expect(fetch).toHaveBeenCalledWith('/api/auth/status', { credentials: 'include' })
        expect(next).toHaveBeenCalledWith()
    })

    it('should redirect unauthenticated user from /session route to home (AC 4)', async () => {
        const to = createMockRoute('/session/abc123')
        const from = createMockRoute('/')

        ; (fetch as any).mockResolvedValueOnce({
            json: async () => ({ authenticated: false })
        })

        await authGuard(to, from, next)

        expect(next).toHaveBeenCalledWith({
            path: '/',
            query: { redirect: '/session/abc123' }
        })
    })

    it('should redirect unauthenticated user from /join route with invite preserved (AC 5)', async () => {
        const to = createMockRoute('/join/INVITE123', { token: 'INVITE123' })
        const from = createMockRoute('/')

        ; (fetch as any).mockResolvedValueOnce({
            json: async () => ({ authenticated: false })
        })

        await authGuard(to, from, next)

        expect(next).toHaveBeenCalledWith({
            path: '/',
            query: { invite: 'INVITE123' }
        })
    })

    it('should handle fetch errors gracefully', async () => {
        const to = createMockRoute('/session/abc123')
        const from = createMockRoute('/')

        ; (fetch as any).mockRejectedValueOnce(new Error('Network error'))

        await authGuard(to, from, next)

        expect(next).toHaveBeenCalledWith({
            path: '/',
            query: { redirect: '/session/abc123' }
        })
    })
})
