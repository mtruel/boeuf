import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePlayerStore } from '../player'

// Mock fetch
global.fetch = vi.fn()

describe('Player Store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
    })

    afterEach(() => {
        vi.resetAllMocks()
    })

    it('initializes with default state', () => {
        const store = usePlayerStore()

        expect(store.isPlaying).toBe(false)
        expect(store.positionMs).toBe(0)
        expect(store.track).toBeNull()
        expect(store.hasTrack).toBe(false)
        expect(store.isLoading).toBe(false)
        expect(store.error).toBeNull()
    })

    it('init() sets sessionId and fetches initial state', async () => {
        const store = usePlayerStore()
        
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => ({
                state: {
                    isPlaying: true,
                    positionMs: 30000,
                    track: {
                        trackId: 'track_1',
                        trackName: 'Test Track',
                        artist: 'Test Artist',
                        durationMs: 180000
                    }
                }
            })
        })

        await store.init('sess_123')

        expect(store.sessionId).toBe('sess_123')
        expect(store.isPlaying).toBe(true)
        expect(store.positionMs).toBeGreaterThanOrEqual(30000)
        expect(store.positionMs).toBeLessThanOrEqual(30010)
        expect(store.track?.name).toBe('Test Track')
    })

    it('reset() clears all state', () => {
        const store = usePlayerStore()
        store.init('sess_123')
        store.isPlaying = true
        store.lastServerPositionMs = 12345
        store.lastServerUpdateAt = new Date()
        store.track = {
            id: 'track_1',
            name: 'Test Track',
            artist: 'Test Artist',
            durationMs: 180000
        }

        store.reset()

        expect(store.isPlaying).toBe(false)
        expect(store.positionMs).toBe(0)
        expect(store.track).toBeNull()
        expect(store.sessionId).toBeNull()
    })

    it('pausePlayer() sends POST request with clientMsgId', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123' // Set directly instead of calling init

        const mockResponse = { eventSeq: 42, cached: false }
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockResponse
        })

        await store.pausePlayer()

        expect(fetch).toHaveBeenCalledWith(
            '/api/sessions/sess_123/player/pause',
            expect.objectContaining({
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: expect.stringContaining('clientMsgId')
            })
        )

        // No optimistic update - state stays same until WS event
        expect(store.isPlaying).toBe(false)
        expect(store.error).toBeNull()
    })

    it('resumePlayer() sends POST request', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123'

        const mockResponse = { eventSeq: 43, cached: false }
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockResponse
        })

        await store.resumePlayer()

        expect(fetch).toHaveBeenCalledWith(
            '/api/sessions/sess_123/player/resume',
            expect.objectContaining({
                method: 'POST',
                credentials: 'include'
            })
        )

        // No optimistic update
        expect(store.isPlaying).toBe(false)
    })

    it('nextTrack() sends POST request', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123'

        const mockResponse = { eventSeq: 44 }
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockResponse
        })

        await store.nextTrack()

        expect(fetch).toHaveBeenCalledWith(
            '/api/sessions/sess_123/player/next',
            expect.objectContaining({
                method: 'POST'
            })
        )
    })

    it('seekTo() sends POST request with positionMs', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123'

        const mockResponse = { eventSeq: 45 }
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: async () => mockResponse
        })

        await store.seekTo(60000) // 1 minute

        expect(fetch).toHaveBeenCalledWith(
            '/api/sessions/sess_123/player/seek',
            expect.objectContaining({
                method: 'POST',
                body: expect.stringContaining('"positionMs":60000')
            })
        )

        // Optimistic update applies immediately
        expect(store.positionMs).toBe(60000)
    })

    it('seekTo() rejects negative position', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123'

        // Clear any fetch calls
        global.fetch = vi.fn()

        await store.seekTo(-100)

        expect(fetch).not.toHaveBeenCalled()
        expect(store.error).toBe('Invalid position')
    })

    it('handles API error gracefully', async () => {
        const store = usePlayerStore()
        store.sessionId = 'sess_123'

        global.fetch = vi.fn().mockResolvedValue({
            ok: false,
            json: async () => ({ code: 'SPOTIFY_NO_DEVICE', message: 'No device' })
        })

        await store.pausePlayer()

        expect(store.error).toBe('SPOTIFY_NO_DEVICE')
    })

    it('getFriendlyErrorMessage() returns user-friendly messages', () => {
        const store = usePlayerStore()

        expect(store.getFriendlyErrorMessage('SPOTIFY_RATE_LIMITED'))
            .toContain('Spotify is busy')
        expect(store.getFriendlyErrorMessage('SPOTIFY_NO_DEVICE'))
            .toContain('No active Spotify device')
        expect(store.getFriendlyErrorMessage('PARTICIPANT_NOT_SYNCED'))
            .toContain('Start Listening')
        expect(store.getFriendlyErrorMessage('UNKNOWN_ERROR'))
            .toContain('An error occurred')
    })

    it('clearError() resets error state', () => {
        const store = usePlayerStore()
        store.error = 'SOME_ERROR'

        store.clearError()

        expect(store.error).toBeNull()
    })
})
