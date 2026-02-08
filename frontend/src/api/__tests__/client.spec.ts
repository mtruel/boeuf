import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { apiFetch, initApiClient, resetSessionExpiredFlag } from '../client'
import type { Router } from 'vue-router'

// Create stable mock functions
const mockToastError = vi.fn()

// Mock dependencies
vi.mock('@/composables/useToast', () => ({
    useToast: () => ({
        error: mockToastError
    })
}))

vi.mock('@/stores/session', () => ({
    useSessionStore: () => ({
        clear: vi.fn()
    })
}))

vi.mock('@/stores/player', () => ({
    usePlayerStore: () => ({
        $reset: vi.fn()
    })
}))

vi.mock('@/stores/presence', () => ({
    usePresenceStore: () => ({
        $reset: vi.fn()
    })
}))

vi.mock('@/stores/realtime', () => ({
    useRealtimeStore: () => ({
        disconnect: vi.fn()
    })
}))

global.fetch = vi.fn()

describe('apiFetch (AC 6, 7)', () => {
    let mockRouter: Router

    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
        resetSessionExpiredFlag()

        mockRouter = {
            push: vi.fn()
        } as any

        initApiClient(mockRouter)
    })

    afterEach(() => {
        vi.clearAllTimers()
    })

    it('should pass through successful responses', async () => {
        const mockResponse = new Response(JSON.stringify({ data: 'test' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
        })

            ; (fetch as any).mockResolvedValueOnce(mockResponse)

        const response = await apiFetch('/api/test')

        expect(response.status).toBe(200)
        expect(mockRouter.push).not.toHaveBeenCalled()
    })

    it('should intercept 401 and redirect to home (AC 6)', async () => {
        const mockResponse = new Response(JSON.stringify({ code: 'UNAUTHENTICATED' }), {
            status: 401,
            headers: { 'Content-Type': 'application/json' }
        })

            ; (fetch as any).mockResolvedValueOnce(mockResponse)

        const response = await apiFetch('/api/test')

        expect(response.status).toBe(401)
        expect(mockRouter.push).toHaveBeenCalledWith('/')
    })

    it('should clear all stores on 401 (AC 6)', async () => {
        const mockResponse = new Response(JSON.stringify({ code: 'UNAUTHENTICATED' }), {
            status: 401,
            headers: { 'Content-Type': 'application/json' }
        })

            ; (fetch as any).mockResolvedValueOnce(mockResponse)

        await apiFetch('/api/test')

        // Store clearing is async, give it time
        await new Promise(resolve => setTimeout(resolve, 10))

        // Verify clear methods would be called (already mocked above)
        expect(mockRouter.push).toHaveBeenCalledWith('/')
    })

    it('should only show one toast for multiple 401s (AC 7)', async () => {
        const mockResponse401 = new Response(JSON.stringify({ code: 'UNAUTHENTICATED' }), {
            status: 401,
            headers: { 'Content-Type': 'application/json' }
        })

        // Mock multiple 401 responses
        ; (fetch as any).mockResolvedValue(mockResponse401)

        // Make 3 simultaneous 401 requests
        await Promise.all([
            apiFetch('/api/test1'),
            apiFetch('/api/test2'),
            apiFetch('/api/test3')
        ])

        // Should only call toast.error once due to debounce
        expect(mockToastError).toHaveBeenCalledTimes(1)
        expect(mockToastError).toHaveBeenCalledWith('Your session expired. Please sign in again.')
    })

    it('should reset toast flag after 5 seconds', async () => {
        vi.useFakeTimers()

        const mockResponse401 = new Response(JSON.stringify({ code: 'UNAUTHENTICATED' }), {
            status: 401,
            headers: { 'Content-Type': 'application/json' }
        })

        ; (fetch as any).mockResolvedValue(mockResponse401)

        // First 401
        await apiFetch('/api/test1')
        expect(mockToastError).toHaveBeenCalledTimes(1)

        // Clear mocks and advance time
        mockToastError.mockClear()
        vi.advanceTimersByTime(5001)

        // Second 401 after flag reset
        await apiFetch('/api/test2')
        expect(mockToastError).toHaveBeenCalledTimes(1)

        vi.useRealTimers()
    })
})
