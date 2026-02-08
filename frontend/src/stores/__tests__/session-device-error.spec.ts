import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '../session'

// Create mock functions
const mockToastCustom = vi.fn()
const mockToastSuccess = vi.fn()

// Mock dependencies
vi.mock('@/composables/useToast', () => ({
    useToast: () => ({
        custom: mockToastCustom,
        success: mockToastSuccess,
        error: vi.fn(),
        warning: vi.fn(),
        info: vi.fn(),
        dismiss: vi.fn()
    })
}))

vi.mock('@/api/client', () => ({
    apiFetch: vi.fn()
}))

vi.mock('./realtime', () => ({
    useRealtimeStore: () => ({
        onMessage: vi.fn(() => vi.fn())
    })
}))

vi.mock('./player', () => ({
    usePlayerStore: () => ({
        init: vi.fn()
    })
}))

describe('SessionStore - Device Error Handling (AC 2, 8)', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
        mockToastCustom.mockClear()
        mockToastSuccess.mockClear()
    })

    it('should show device error toast when SPOTIFY_NO_DEVICE (AC 2)', async () => {
        const store = useSessionStore()
        const { apiFetch } = await import('@/api/client')

        const mockErrorResponse = new Response(
            JSON.stringify({
                code: 'SPOTIFY_NO_DEVICE',
                message: 'No device',
                requiresActiveDevice: true
            }),
            { status: 503 }
        )

        vi.mocked(apiFetch).mockResolvedValueOnce(mockErrorResponse)

        store.sessionId = 'test-session-123'

        await store.startListening()

        // Should stay in ready state for retry
        expect(store.syncState).toBe('ready')
        expect(store.error).toBe('SPOTIFY_NO_DEVICE')

        // Should show custom toast with actions
        expect(mockToastCustom).toHaveBeenCalledWith(
            'No Active Spotify Device',
            expect.stringContaining('Please start playback'),
            expect.objectContaining({
                duration: Infinity,
                action: expect.any(Object),
                cancel: expect.any(Object)
            })
        )
    })

    it('should show success toast on successful sync', async () => {
        const store = useSessionStore()
        const { apiFetch } = await import('@/api/client')

        const mockSuccessResponse = new Response(
            JSON.stringify({ syncState: 'synced', nowPlaying: null }),
            { status: 200 }
        )

        vi.mocked(apiFetch).mockResolvedValueOnce(mockSuccessResponse)

        store.sessionId = 'test-session-123'

        await store.startListening()

        expect(store.syncState).toBe('synced')
        expect(mockToastSuccess).toHaveBeenCalledWith("Synced! You're now listening together.")
    })

    it('should apply backoff on retry (AC 8)', async () => {
        vi.useFakeTimers()

        const store = useSessionStore()
        store.sessionId = 'test-session-123'

        const { apiFetch } = await import('@/api/client')

        const mockSuccessResponse = new Response(
            JSON.stringify({ syncState: 'synced', nowPlaying: null }),
            { status: 200 }
        )

        vi.mocked(apiFetch).mockResolvedValueOnce(mockSuccessResponse)

        const retryPromise = store.retryWithBackoff()

        expect(store.retryCount).toBe(1)
        expect(vi.mocked(apiFetch)).not.toHaveBeenCalled()

        await vi.advanceTimersByTimeAsync(2000)
        await retryPromise

        expect(vi.mocked(apiFetch)).toHaveBeenCalled()
        expect(store.retryCount).toBe(0)
        expect(store.syncState).toBe('synced')

        vi.useRealTimers()
    })
})
