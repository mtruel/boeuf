/**
 * Player Store - Manages playback state and player controls
 * 
 * Handles:
 * - Player state (isPlaying, track, position)
 * - Player actions (pause, resume, next, seek)
 * - WebSocket event processing for PLAYER_* events
 * - Idempotent commands with clientMsgId
 * - Error handling (Spotify rate limits, no device, etc.)
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Ref } from 'vue'
import { useRealtimeStore, type WSMessage, type NowPlayingInfo } from './realtime'

export interface Track {
    id: string
    name: string
    artist: string
    album?: string
    durationMs: number
    imageUrl?: string
}

export interface PlayerState {
    isPlaying: boolean
    positionMs: number
    track: Track | null
    lastUpdate: Date | null
}

interface LoadingState {
    pause: boolean
    resume: boolean
    next: boolean
    seek: boolean
}

/**
 * Player Store
 */
export const usePlayerStore = defineStore('player', () => {
    // State
    const sessionId: Ref<string | null> = ref(null)
    const isPlaying: Ref<boolean> = ref(false)
    const positionMs: Ref<number> = ref(0)
    const currentTrack: Ref<Track | null> = ref(null)
    const track: Ref<Track | null> = currentTrack
    const lastUpdateAt: Ref<Date | null> = ref(null)
    const lastEventSeq: Ref<number> = ref(0)

    const loading: Ref<LoadingState> = ref({
        pause: false,
        resume: false,
        next: false,
        seek: false
    })

    const error: Ref<string | null> = ref(null)

    // Computed
    const hasTrack = computed(() => track.value !== null)
    const progressPercent = computed(() => {
        if (!currentTrack.value || currentTrack.value.durationMs === 0) return 0
        return Math.min(100, (positionMs.value / currentTrack.value.durationMs) * 100)
    })

    const formatMs = (ms: number): string => {
        const totalSec = Math.floor(ms / 1000)
        const minutes = Math.floor(totalSec / 60)
        const seconds = totalSec % 60
        return `${minutes}:${seconds.toString().padStart(2, '0')}`
    }

    const positionFormatted = computed(() => formatMs(positionMs.value))
    const durationFormatted = computed(() => {
        if (!currentTrack.value) return '0:00'
        return formatMs(currentTrack.value.durationMs)
    })
    const isLoading = computed(() =>
        loading.value.pause ||
        loading.value.resume ||
        loading.value.next ||
        loading.value.seek
    )

    /**
     * Initialize player store for a session
     */
    async function init(sid: string) {
        sessionId.value = sid

        // Register WebSocket message handlers
        const realtime = useRealtimeStore()
        realtime.registerHandler('PLAYER_PAUSED', handlePlayerPaused)
        realtime.registerHandler('PLAYER_RESUMED', handlePlayerResumed)
        realtime.registerHandler('TRACK_CHANGED', handleTrackChanged)
        realtime.registerHandler('PLAYER_SEEKED', handlePlayerSeeked)
        realtime.registerHandler('PLAYER_STATE_UPDATE', handlePlayerStateUpdate)

        // Fetch initial player state with retry
        const maxRetries = 3
        let attempt = 0
        while (attempt < maxRetries) {
            try {
                const response = await fetch(`/api/sessions/${sid}/player/state`, {
                    credentials: 'include'
                })

                // If fetch failed or returned non‑OK, treat as no initial state (e.g., during tests)
                if (!response || !response.ok) {
                    // No state available – keep defaults and stop retrying
                    return
                }

                const data = await response.json()
                if (data.state && data.state.track) {
                    // Update store with full initial state
                    isPlaying.value = data.state.isPlaying
                    positionMs.value = data.state.positionMs
                    currentTrack.value = data.state.track
                    lastUpdateAt.value = new Date()
                    return // Success
                } else if (data.state) {
                    // Partial state (no track yet)
                    isPlaying.value = data.state.isPlaying
                    positionMs.value = data.state.positionMs
                    currentTrack.value = null
                    lastUpdateAt.value = new Date()
                    return // Success
                }
            } catch (err) {
                attempt++
                if (attempt < maxRetries) {
                    const backoff = Math.pow(2, attempt - 1) * 100 // 100ms, 200ms, 400ms
                    await new Promise(resolve => setTimeout(resolve, backoff))
                } else {
                    error.value = 'INIT_FAILED'
                    console.error('Failed to fetch initial player state after retries:', err)
                }
            }
        }
    }

    /**
     * Reset player state (on disconnect/error)
     */
    function reset() {
        isPlaying.value = false
        positionMs.value = 0
        currentTrack.value = null
        lastUpdateAt.value = null
        lastEventSeq.value = 0
        error.value = null
        sessionId.value = null
    }

    /**
     * Pause playback
     */
    async function pausePlayer() {
        if (!sessionId.value) {
            error.value = 'No active session'
            return
        }

        loading.value.pause = true
        error.value = null

        // Retry logic for device activation
        const maxRetries = 3
        const retryDelays = [2000, 4000, 8000] // 2s, 4s, 8s exponential backoff

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const clientMsgId = crypto.randomUUID()
                const response = await fetch(`/api/sessions/${sessionId.value}/player/pause`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    credentials: 'include',
                    body: JSON.stringify({ clientMsgId })
                })

                if (!response.ok) {
                    const errorData = await response.json()
                    const errorCode = errorData.code || 'PAUSE_FAILED'

                    // Retry on device not active
                    if (errorCode === 'SPOTIFY_NO_ACTIVE_DEVICE' && attempt < maxRetries - 1) {
                        console.warn(`Device not active, retrying in ${retryDelays[attempt]}ms (attempt ${attempt + 1}/${maxRetries})`)
                        await new Promise(resolve => setTimeout(resolve, retryDelays[attempt]))
                        continue
                    }

                    throw new Error(errorCode)
                }

                const data = await response.json()
                console.log('Pause command sent, eventSeq:', data.eventSeq)
                // State will be updated by WS event from server
                return // Success, exit retry loop

            } catch (err: any) {
                if (attempt === maxRetries - 1) {
                    // Final attempt failed
                    error.value = err.message
                    console.error('Failed to pause player after retries:', err)
                }
            } finally {
                if (attempt === maxRetries - 1) {
                    loading.value.pause = false
                }
            }
        }
    }

    /**
     * Resume playback
     */
    async function resumePlayer() {
        if (!sessionId.value) {
            error.value = 'No active session'
            return
        }

        loading.value.resume = true
        error.value = null

        // Retry logic for device activation
        const maxRetries = 3
        const retryDelays = [2000, 4000, 8000]

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const clientMsgId = crypto.randomUUID()
                const response = await fetch(`/api/sessions/${sessionId.value}/player/resume`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    credentials: 'include',
                    body: JSON.stringify({ clientMsgId })
                })

                if (!response.ok) {
                    const errorData = await response.json()
                    const errorCode = errorData.code || 'RESUME_FAILED'

                    // Retry on device not active
                    if (errorCode === 'SPOTIFY_NO_ACTIVE_DEVICE' && attempt < maxRetries - 1) {
                        console.warn(`Device not active, retrying in ${retryDelays[attempt]}ms (attempt ${attempt + 1}/${maxRetries})`)
                        await new Promise(resolve => setTimeout(resolve, retryDelays[attempt]))
                        continue
                    }

                    throw new Error(errorCode)
                }

                const data = await response.json()
                console.log('Resume command sent, eventSeq:', data.eventSeq)
                return

            } catch (err: any) {
                if (attempt === maxRetries - 1) {
                    error.value = err.message
                    console.error('Failed to resume player after retries:', err)
                }
            } finally {
                if (attempt === maxRetries - 1) {
                    loading.value.resume = false
                }
            }
        }
    }

    /**
     * Skip to next track
     */
    async function nextTrack() {
        if (!sessionId.value) {
            error.value = 'No active session'
            return
        }

        loading.value.next = true
        error.value = null

        try {
            const clientMsgId = crypto.randomUUID()
            const response = await fetch(`/api/sessions/${sessionId.value}/player/next`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ clientMsgId })
            })

            if (!response.ok) {
                const errorData = await response.json()
                throw new Error(errorData.code || 'NEXT_FAILED')
            }

            const data = await response.json()
            console.log('Next command sent, eventSeq:', data.eventSeq)

        } catch (err: any) {
            error.value = err.message
            console.error('Failed to skip track:', err)
        } finally {
            loading.value.next = false
        }
    }

    /**
     * Seek to position in track
     */
    async function seekTo(targetPositionMs: number) {
        if (!sessionId.value) {
            error.value = 'No active session'
            return
        }

        if (targetPositionMs < 0) {
            error.value = 'Invalid position'
            return
        }

        loading.value.seek = true
        error.value = null

        try {
            const clientMsgId = crypto.randomUUID()
            const response = await fetch(`/api/sessions/${sessionId.value}/player/seek`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ clientMsgId, positionMs: targetPositionMs })
            })

            if (!response.ok) {
                const errorData = await response.json()
                throw new Error(errorData.code || 'SEEK_FAILED')
            }

            const data = await response.json()
            console.log('Seek command sent, eventSeq:', data.eventSeq)
            // State will be updated by WS event from server

        } catch (err: any) {
            error.value = err.message
            console.error('Failed to seek:', err)
        } finally {
            loading.value.seek = false
        }
    }

    /**
     * WebSocket event handlers
     */

    function handlePlayerPaused(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = false
        positionMs.value = payload.positionMs || 0

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        console.log('Player paused by', payload.userId)
    }

    function handlePlayerResumed(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = true
        positionMs.value = payload.positionMs || 0

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        console.log('Player resumed by', payload.userId)
    }

    function handleTrackChanged(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = payload.isPlaying ?? true
        positionMs.value = payload.positionMs || 0

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        console.log('Track changed by', payload.userId)
    }

    function handlePlayerSeeked(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = payload.isPlaying ?? isPlaying.value
        positionMs.value = payload.positionMs || 0

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        console.log('Player seeked by', payload.userId)
    }

    function handlePlayerStateUpdate(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any

        // Sync position from server: recalibrate if drift > 2s
        const drift = Math.abs(positionMs.value - (payload.positionMs || 0))
        if (drift > 2000) {
            positionMs.value = payload.positionMs || 0
        }

        // Update playing state and last update timestamp
        isPlaying.value = payload.isPlaying ?? isPlaying.value
        lastUpdateAt.value = new Date()

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        console.log('Player state updated from polling:', payload)
    }

    /**
     * Helper: Check for eventSeq gaps (missing messages)
     */
    function checkEventSeq(eventSeq: number) {
        if (lastEventSeq.value > 0 && eventSeq !== lastEventSeq.value + 1) {
            console.warn(`EventSeq gap detected: expected ${lastEventSeq.value + 1}, got ${eventSeq}`)
            // Future: trigger resync (Story 4.3)
        }
        lastEventSeq.value = eventSeq
    }

    /**
     * Helper: Update track info
     */
    function updateTrack(trackData: any) {
        track.value = {
            id: trackData.trackId || '',
            name: trackData.trackName || 'Unknown',
            artist: trackData.artist || 'Unknown Artist',
            album: trackData.album,
            durationMs: trackData.durationMs || 0,
            imageUrl: trackData.imageUrl
        }
    }

    /**
     * Get user-friendly error message
     */
    function getFriendlyErrorMessage(code: string): string {
        const messages: Record<string, string> = {
            'SPOTIFY_RATE_LIMITED': 'Spotify is busy. Please try again in a moment.',
            'SPOTIFY_NO_DEVICE': 'No active Spotify device found. Please start Spotify.',
            'SPOTIFY_NO_ACTIVE_DEVICE': 'No active Spotify device. Start playback in Spotify and try again.',
            'SPOTIFY_UNAVAILABLE': 'Unable to control playback. Please check your connection.',
            'PARTICIPANT_NOT_SYNCED': 'Please click "Start Listening" first.',
        }
        return messages[code] || 'An error occurred. Please try again.'
    }

    /**
     * Clear error
     */
    function clearError() {
        error.value = null
    }

    /**
     * Generic player state updater used by WS handlers.
     * Updates isPlaying, positionMs, lastUpdateAt and optionally track metadata.
     */
    function updatePlayerState(payload: any) {
        isPlaying.value = payload.isPlaying ?? isPlaying.value
        positionMs.value = payload.positionMs || 0
        lastUpdateAt.value = new Date()
        if (payload.track) {
            updateTrack(payload.track)
        }
    }

    /**
     * Action to update only track metadata (e.g., when TRACK_CHANGED provides new track info).
     */
    function updateTrackMetadata(trackData: any) {
        updateTrack(trackData)
    }

    return {
        // State
        sessionId,
        isPlaying,
        positionMs,
        track,
        currentTrack,
        lastUpdateAt,
        loading,
        error,
        hasTrack,
        isLoading,
        progressPercent,
        positionFormatted,
        durationFormatted,

        // Actions
        init,
        reset,
        pausePlayer,
        resumePlayer,
        nextTrack,
        seekTo,
        getFriendlyErrorMessage,
        clearError,
        updatePlayerState,
        updateTrackMetadata
    }
})
