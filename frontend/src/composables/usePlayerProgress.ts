/**
 * usePlayerProgress - Composable for local interpolation of player progress
 *
 * Handles:
 * - Local interpolation of playback position via requestAnimationFrame
 * - Automatic increment every second when isPlaying === true
 * - Recalibration when server drift exceeds threshold (>2s)
 * - Pause state handling (freezes interpolation)
 * - Cleanup on unmount
 *
 * AC 2, 5, 8: Real-time progress bar with smooth local interpolation
 */

import { ref, computed, watch, onUnmounted, readonly } from 'vue'
import { usePlayerStore } from '@/stores/player'

const DRIFT_THRESHOLD_MS = 2000 // Recalibrate if server drift > 2s
const UPDATE_INTERVAL_MS = 1000 // Update every second

/**
 * Format milliseconds to MM:SS
 */
export const formatTime = (ms: number): string => {
    if (typeof ms !== 'number' || isNaN(ms)) return '0:00'
    const totalSeconds = Math.floor(ms / 1000)
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

export function usePlayerProgress() {
    const playerStore = usePlayerStore()

    // Local interpolated position
    const interpolatedPositionMs = ref(0)
    const lastServerPositionMs = ref(0)
    const lastServerUpdateAt = ref<number | null>(null)
    const isPlaying = ref(false)

    // Animation frame ID for cleanup
    let animationFrameId: number | null = null
    let intervalId: number | null = null

    /**
     * Current interpolated position in milliseconds
     */
    const currentPositionMs = computed(() => {
        return Math.round(interpolatedPositionMs.value)
    })

    /**
     * Current position formatted as MM:SS
     */
    const currentPositionFormatted = computed(() => {
        return formatTime(currentPositionMs.value)
    })

    /**
     * Progress percentage (0-100)
     */
    const progressPercent = computed(() => {
        const duration = playerStore.currentTrack?.durationMs || 0
        if (duration === 0) return 0
        return Math.min(100, (currentPositionMs.value / duration) * 100)
    })

    /**
     * Start local interpolation
     */
    const startInterpolation = () => {
        // Clear any existing interval
        if (intervalId !== null) {
            window.clearInterval(intervalId)
        }

        // Initialize from current store values
        const storePosition = playerStore.positionMs || 0
        lastServerPositionMs.value = storePosition
        lastServerUpdateAt.value = Date.now()
        isPlaying.value = playerStore.isPlaying
        interpolatedPositionMs.value = storePosition

        if (!isPlaying.value) return

        // Update position every second
        intervalId = window.setInterval(() => {
            if (!isPlaying.value) return

            const duration = playerStore.currentTrack?.durationMs || 0
            const newPosition = interpolatedPositionMs.value + UPDATE_INTERVAL_MS

            // Cap at duration
            if (duration > 0 && newPosition >= duration) {
                interpolatedPositionMs.value = duration
            } else {
                interpolatedPositionMs.value = newPosition
            }
        }, UPDATE_INTERVAL_MS)
    }

    /**
     * Stop local interpolation
     */
    const stopInterpolation = () => {
        if (intervalId !== null) {
            window.clearInterval(intervalId)
            intervalId = null
        }
        if (animationFrameId !== null) {
            cancelAnimationFrame(animationFrameId)
            animationFrameId = null
        }
    }

    /**
     * Recalibrate position from server if drift exceeds threshold
     */
    const recalibrateIfNeeded = (serverPositionMs: number) => {
        const serverPos = serverPositionMs || 0
        const drift = Math.abs(interpolatedPositionMs.value - serverPos)

        if (drift > DRIFT_THRESHOLD_MS) {
            // Smooth recalibration - just update to server position
            interpolatedPositionMs.value = serverPos
            lastServerPositionMs.value = serverPos
            lastServerUpdateAt.value = Date.now()
        }
    }

    /**
     * Watch for changes in player store
     */
    // Watch isPlaying changes
    watch(
        () => playerStore.isPlaying,
        (newIsPlaying) => {
            isPlaying.value = newIsPlaying

            if (newIsPlaying) {
                // Resume interpolation from current position
                startInterpolation()
            } else {
                // Pause interpolation - freeze at current position
                stopInterpolation()
                // Sync with server position when paused
                interpolatedPositionMs.value = playerStore.positionMs || 0
            }
        },
        { immediate: true }
    )

    // Watch position changes from server
    watch(
        () => playerStore.positionMs,
        (newPositionMs) => {
            const pos = newPositionMs || 0
            if (!isPlaying.value) {
                // When paused, always sync with server
                interpolatedPositionMs.value = pos
            } else {
                // When playing, recalibrate only if drift exceeds threshold
                recalibrateIfNeeded(pos)
            }
            lastServerPositionMs.value = pos
            lastServerUpdateAt.value = Date.now()
        }
    )

    // Watch track changes - reset interpolation
    watch(
        () => playerStore.currentTrack?.id,
        (newTrackId, oldTrackId) => {
            if (newTrackId !== oldTrackId) {
                // Track changed - reset to server position
                const pos = playerStore.positionMs || 0
                interpolatedPositionMs.value = pos
                lastServerPositionMs.value = pos
                lastServerUpdateAt.value = Date.now()
            }
        }
    )

    // Cleanup on unmount
    onUnmounted(() => {
        stopInterpolation()
    })

    return {
        currentPositionMs,
        currentPositionFormatted,
        progressPercent,
        interpolatedPositionMs: readonly(interpolatedPositionMs),
        lastServerPositionMs: readonly(lastServerPositionMs),
        startInterpolation,
        stopInterpolation,
        recalibrateIfNeeded
    }
}
