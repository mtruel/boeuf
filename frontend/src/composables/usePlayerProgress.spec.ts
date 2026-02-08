/**
 * Tests for usePlayerProgress composable
 *
 * AC 2, 5, 8: Local interpolation of player progress
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { usePlayerProgress, formatTime } from './usePlayerProgress'
import { setActivePinia, createPinia } from 'pinia'
import { usePlayerStore, type Track } from '@/stores/player'

describe('usePlayerProgress', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        vi.useFakeTimers({ shouldAdvanceTime: true })
    })

    afterEach(() => {
        vi.restoreAllMocks()
        vi.useRealTimers()
    })

    it('should initialize with position from store', () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: false,
            positionMs: 125000, // 2:05
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        expect(progress.currentPositionMs.value).toBe(125000)
        expect(progress.currentPositionFormatted.value).toBe('2:05')
    })

    it('AC 8: should increment position every second when playing', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 251000, // 4:11
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 600000 }
        })

        const progress = usePlayerProgress()

        // Initial position
        expect(progress.currentPositionFormatted.value).toBe('4:11')

        // Advance 1 second
        vi.advanceTimersByTime(1000)
        await nextTick()

        expect(progress.currentPositionFormatted.value).toBe('4:12')

        // Advance 9 more seconds (total 10s)
        vi.advanceTimersByTime(9000)
        await nextTick()

        expect(progress.currentPositionFormatted.value).toBe('4:21')
    })

    it('AC 8: should freeze position when paused', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 60000, // 1:00
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        // Advance 2 seconds while playing
        vi.advanceTimersByTime(2000)
        await nextTick()

        expect(progress.currentPositionFormatted.value).toBe('1:02')

        // Pause - server position has advanced to 1:01 (simulating real-world scenario)
        store.positionMs = 62000 // 1:02 - server has caught up
        store.isPlaying = false
        await nextTick()

        // Advance 5 more seconds while paused
        vi.advanceTimersByTime(5000)
        await nextTick()

        // Position should remain frozen at server-synced position
        expect(progress.currentPositionFormatted.value).toBe('1:02')
    })

    it('AC 5: should recalibrate when server drift exceeds 2s threshold', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 60000, // 1:00
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        // Advance 5 seconds locally
        vi.advanceTimersByTime(5000)
        await nextTick()

        // Local position should be 1:05
        expect(progress.currentPositionFormatted.value).toBe('1:05')

        // Server sends position that's 3 seconds behind (drift > 2s threshold)
        store.positionMs = 62000 // 1:02 (3s behind local)
        await nextTick()

        // Should recalibrate to server position
        expect(progress.currentPositionFormatted.value).toBe('1:02')
    })

    it('AC 5: should NOT recalibrate when server drift is within 2s threshold', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 60000, // 1:00
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        // Advance 1 second locally
        vi.advanceTimersByTime(1000)
        await nextTick()

        // Local position should be 1:01
        expect(progress.currentPositionFormatted.value).toBe('1:01')

        // Server sends position that's 0.5s behind (drift < 2s threshold)
        store.positionMs = 60500 // 1:00.5
        await nextTick()

        // Should NOT recalibrate - keep local position
        expect(progress.currentPositionFormatted.value).toBe('1:01')
    })

    it('should reset position when track changes', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 120000, // 2:00
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        // Advance some time
        vi.advanceTimersByTime(5000)
        await nextTick()

        expect(progress.currentPositionFormatted.value).toBe('2:05')

        // Track changes - server sends new position
        store.positionMs = 5000 // 0:05
        store.currentTrack = { id: 'track2', name: 'New Track', artist: 'Artist', durationMs: 200000 }
        await nextTick()

        // Position should reset to new track position
        expect(progress.currentPositionFormatted.value).toBe('0:05')
    })

    it('should cap position at track duration', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 58000, // 0:58
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 60000 } // 1:00 duration
        })

        const progress = usePlayerProgress()

        // Advance 5 seconds - should cap at duration
        vi.advanceTimersByTime(5000)
        await nextTick()

        // Should be capped at 1:00 (duration)
        expect(progress.currentPositionFormatted.value).toBe('1:00')
        expect(progress.currentPositionMs.value).toBe(60000)
    })

    it('should calculate progress percent correctly', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: true,
            positionMs: 60000, // 1:00
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 120000 } // 2:00
        })

        const progress = usePlayerProgress()

        expect(progress.progressPercent.value).toBe(50)

        // Advance 30 seconds
        vi.advanceTimersByTime(30000)
        await nextTick()

        expect(progress.progressPercent.value).toBe(75)
    })

    it('should return 0% progress when no track', () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: false,
            positionMs: 0,
            currentTrack: null
        })

        const progress = usePlayerProgress()

        expect(progress.progressPercent.value).toBe(0)
    })

    it('should format time correctly', () => {
        expect(formatTime(0)).toBe('0:00')
        expect(formatTime(59000)).toBe('0:59')
        expect(formatTime(60000)).toBe('1:00')
        expect(formatTime(123456)).toBe('2:03')
        expect(formatTime(NaN)).toBe('0:00')
    })

    it('should sync with server position when resuming playback', async () => {
        const store = usePlayerStore()
        store.$patch({
            isPlaying: false,
            positionMs: 30000, // 0:30
            currentTrack: { id: 'track1', name: 'Test', artist: 'Artist', durationMs: 300000 }
        })

        const progress = usePlayerProgress()

        expect(progress.currentPositionFormatted.value).toBe('0:30')

        // Server updates position while paused
        store.positionMs = 45000 // 0:45
        await nextTick()

        // Should sync immediately when paused
        expect(progress.currentPositionFormatted.value).toBe('0:45')

        // Resume playback
        store.isPlaying = true
        await nextTick()

        // Advance 2 seconds
        vi.advanceTimersByTime(2000)
        await nextTick()

        // Should continue from synced position
        expect(progress.currentPositionFormatted.value).toBe('0:47')
    })
})
