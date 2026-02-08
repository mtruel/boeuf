import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '../session'

// Mock fetch globally
global.fetch = vi.fn()

vi.mock('@/stores/player', () => ({
    usePlayerStore: () => ({
        init: vi.fn()
    })
}))

describe('Session Store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
    })

    it('should initialize with ready state', () => {
        const store = useSessionStore()

        expect(store.syncState).toBe('ready')
        expect(store.isReady).toBe(true)
        expect(store.isSynced).toBe(false)
        expect(store.error).toBe(null)
    })

    it('should initialize session correctly', () => {
        const store = useSessionStore()
        const sessionId = 'test-session-123'
        const userId = 'test-user-456'

        store.initialize(sessionId, userId)

        expect(store.sessionId).toBe(sessionId)
    })

    it('should load sync state from server', async () => {
        const store = useSessionStore()
        store.initialize('test-session', 'test-user')

        // Mock successful API response
        global.fetch = vi.fn().mockResolvedValueOnce({
            ok: true,
            json: async () => ({ syncState: 'synced' })
        } as Response)

        await store.loadSyncState()

        expect(store.syncState).toBe('synced')
        expect(store.isSynced).toBe(true)
    })

    it('should start listening successfully', async () => {
        const store = useSessionStore()
        store.initialize('test-session', 'test-user')

        // Mock successful sync/start response
        global.fetch = vi.fn().mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                syncState: 'synced',
                nowPlaying: {
                    trackId: 'spotify:track:123',
                    trackName: 'Test Song',
                    artist: 'Test Artist',
                    isPlaying: true,
                    positionMs: 1000,
                    durationMs: 200000
                }
            })
        } as Response)

        await store.startListening()

        expect(store.syncState).toBe('synced')
        expect(store.nowPlaying).not.toBe(null)
        expect(store.nowPlaying?.trackName).toBe('Test Song')
    })

    it('should handle start listening error', async () => {
        const store = useSessionStore()
        store.initialize('test-session', 'test-user')

        // Mock error response
        global.fetch = vi.fn().mockResolvedValueOnce({
            ok: false,
            status: 409,
            json: async () => ({ code: 'SPOTIFY_NOT_CONNECTED' })
        } as Response)

        await store.startListening()

        expect(store.syncState).toBe('error')
        expect(store.hasError).toBe(true)
        expect(store.error).toBe('SPOTIFY_NOT_CONNECTED')
    })

    it('should clear session state', () => {
        const store = useSessionStore()
        store.initialize('test-session', 'test-user')
        store.syncState = 'synced'

        store.clear()

        expect(store.sessionId).toBe(null)
        expect(store.syncState).toBe('ready')
        expect(store.error).toBe(null)
        expect(store.nowPlaying).toBe(null)
    })

    it('should retry after error', async () => {
        vi.useFakeTimers()

        const store = useSessionStore()
        store.initialize('test-session', 'test-user')
        store.syncState = 'error'
        store.error = 'Test error'

        const client = await import('@/api/client')
        const mockSuccessResponse = new Response(
            JSON.stringify({ syncState: 'synced', nowPlaying: null }),
            { status: 200 }
        )
        const apiFetchSpy = vi
            .spyOn(client, 'apiFetch')
            .mockResolvedValue(mockSuccessResponse)

        const retryPromise = store.retry()

        await vi.advanceTimersByTimeAsync(2000)
        await retryPromise

        expect(apiFetchSpy).toHaveBeenCalled()
        expect(store.syncState).toBe('synced')
        expect(store.error).toBe(null)

        vi.useRealTimers()
    })

    // AC#3: Refresh sans restart - persistance d'état
    describe('AC#3: Refresh without restart (syncState persistence)', () => {
        beforeEach(() => {
            sessionStorage.clear()
        })

        it('should persist syncState to sessionStorage on startListening', async () => {
            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            const client = await import('@/api/client')
            vi.spyOn(client, 'apiFetch').mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        syncState: 'synced',
                        nowPlaying: {
                            trackId: 'spotify:track:123',
                            trackName: 'Test Song',
                            artist: 'Test Artist',
                            isPlaying: true,
                            positionMs: 0,
                            durationMs: 180000
                        }
                    }),
                    { status: 200 }
                )
            )

            await store.startListening()

            const cached = sessionStorage.getItem('boeuf_syncState_session-123')
            expect(cached).toBe('synced')
        })

        it('should restore syncState from sessionStorage on loadSyncState', async () => {
            sessionStorage.setItem('boeuf_syncState_session-123', 'synced')

            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            global.fetch = vi.fn().mockResolvedValueOnce({
                ok: true,
                json: async () => ({ syncState: 'synced' })
            } as Response)

            await store.loadSyncState()

            expect(store.syncState).toBe('synced')
            expect(store.isSynced).toBe(true)
        })

        it('should verify server state takes precedence over cache', async () => {
            sessionStorage.setItem('boeuf_syncState_session-123', 'synced')

            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            const client = await import('@/api/client')
            vi.spyOn(client, 'apiFetch')
                .mockResolvedValueOnce(
                    new Response(JSON.stringify({ syncState: 'ready' }), { status: 200 })
                )
                .mockResolvedValueOnce(
                    new Response(
                        JSON.stringify({ sessionId: 'session-123', nowPlaying: null }),
                        { status: 200 }
                    )
                )

            await store.loadSyncState()

            expect(store.syncState).toBe('ready')
            expect(store.isReady).toBe(true)
        })

        it('should not return to Sas after refresh if was synced', async () => {
            // Simulate user synced and refreshing
            sessionStorage.setItem('boeuf_syncState_session-123', 'synced')

            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            global.fetch = vi.fn().mockResolvedValueOnce({
                ok: true,
                json: async () => ({ syncState: 'synced' })
            } as Response)

            await store.loadSyncState()

            expect(store.isSynced).toBe(true)
            expect(store.isReady).toBe(false)
        })

        it('should handle multiple session IDs with separate cache keys', async () => {
            sessionStorage.setItem('boeuf_syncState_session-1', 'synced')
            sessionStorage.setItem('boeuf_syncState_session-2', 'ready')

            const store1 = useSessionStore()
            store1.initialize('session-1', 'user-1')
            global.fetch = vi.fn().mockResolvedValueOnce({
                ok: true,
                json: async () => ({ syncState: 'synced' })
            } as Response)
            await store1.loadSyncState()
            expect(store1.syncState).toBe('synced')

            store1.clear()
            setActivePinia(createPinia())

            const store2 = useSessionStore()
            store2.initialize('session-2', 'user-2')
            global.fetch = vi.fn().mockResolvedValueOnce({
                ok: true,
                json: async () => ({ syncState: 'ready' })
            } as Response)
            await store2.loadSyncState()
            expect(store2.syncState).toBe('ready')
        })

        it('should clear sessionStorage on clear()', () => {
            sessionStorage.setItem('boeuf_syncState_session-123', 'synced')
            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            store.clear()

            expect(sessionStorage.getItem('boeuf_syncState_session-123')).toBeNull()
        })

        it('should handle API errors gracefully', async () => {
            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            global.fetch = vi.fn().mockResolvedValueOnce({
                ok: false,
                status: 500
            } as Response)

            await store.loadSyncState()

            // Should not crash, keep default ready state
            expect(store.syncState).toBe('ready')
        })

        it('should handle network errors gracefully', async () => {
            const store = useSessionStore()
            store.initialize('session-123', 'user-456')

            global.fetch = vi.fn().mockRejectedValueOnce(new Error('Network error'))

            await store.loadSyncState()

            // Should not crash
            expect(store.syncState).toBe('ready')
        })
    })
})
