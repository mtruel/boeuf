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
import { apiFetch } from '@/api/client'

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
    // Server-authoritative position state (base values for computation)
    const lastServerPositionMs: Ref<number> = ref(0)
    const lastServerUpdateAt: Ref<Date | null> = ref(null)
    const currentTrack: Ref<Track | null> = ref(null)
    const track: Ref<Track | null> = currentTrack
    const lastEventSeq: Ref<number> = ref(0)
    const lastCommand: Ref<{ type: 'pause' | 'resume'; at: number } | null> = ref(null)
    const unregisterHandlers: Array<() => void> = []
    let pollIntervalId: number | null = null
    let pollInFlight = false

    const loading: Ref<LoadingState> = ref({
        pause: false,
        resume: false,
        next: false,
        seek: false
    })

    const error: Ref<string | null> = ref(null)

    // Computed position based on server state + elapsed time when playing
    const positionMs = computed(() => {
        if (!lastServerUpdateAt.value) return lastServerPositionMs.value
        
        const basePosition = lastServerPositionMs.value
        const baseTime = lastServerUpdateAt.value.getTime()
        const now = Date.now()
        
        if (!isPlaying.value) {
            return basePosition
        }
        
        const elapsed = now - baseTime
        const duration = currentTrack.value?.durationMs || 0
        const computed = basePosition + elapsed
        
        // Clamp to duration if track has ended
        if (duration > 0 && computed >= duration) {
            return duration
        }
        
        return computed
    })

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

    const commandClearWindowMs = 4000

    /**
     * Fetch player state from server and update store
     */
    async function fetchPlayerState(maxRetries = 1, setErrorOnFail = false) {
        if (!sessionId.value) return false

        let attempt = 0
        while (attempt < maxRetries) {
            try {
                const response = await apiFetch(`/api/sessions/${sessionId.value}/player/state`, {
                    credentials: 'include'
                })

                if (!response || !response.ok) {
                    return false
                }

                const data = await response.json()
                if (data.state && data.state.track) {
                    isPlaying.value = data.state.isPlaying
                    lastServerPositionMs.value = data.state.positionMs
                    lastServerUpdateAt.value = new Date()
                    updateTrack(data.state.track)
                    return true
                }

                if (data.state) {
                    isPlaying.value = data.state.isPlaying
                    lastServerPositionMs.value = data.state.positionMs
                    lastServerUpdateAt.value = new Date()
                    currentTrack.value = null
                    return true
                }

                return false
            } catch (err) {
                attempt++
                if (attempt < maxRetries) {
                    const backoff = Math.pow(2, attempt - 1) * 100
                    await new Promise(resolve => setTimeout(resolve, backoff))
                } else if (setErrorOnFail) {
                    error.value = 'INIT_FAILED'
                    console.error('Failed to fetch initial player state after retries:', err)
                }
            }
        }

        return false
    }

    /**
     * Refresh player state on demand (single attempt, no error surface)
     */
    async function refreshPlayerState() {
        return fetchPlayerState(1, false)
    }

    function startPolling() {
        if (pollIntervalId !== null) return
        pollIntervalId = window.setInterval(async () => {
            if (pollInFlight) return
            pollInFlight = true
            try {
                await refreshPlayerState()
            } finally {
                pollInFlight = false
            }
        }, 3000)
    }

    function stopPolling() {
        if (pollIntervalId !== null) {
            window.clearInterval(pollIntervalId)
            pollIntervalId = null
        }
        pollInFlight = false
    }

    /**
     * Initialize player store for a session
     */
    async function init(sid: string) {
        sessionId.value = sid

        // Register WebSocket message handlers
        const realtime = useRealtimeStore()
        clearHandlers()
        unregisterHandlers.push(realtime.registerHandler('PLAYER_PAUSED', handlePlayerPaused))
        unregisterHandlers.push(realtime.registerHandler('PLAYER_RESUMED', handlePlayerResumed))
        unregisterHandlers.push(realtime.registerHandler('TRACK_CHANGED', handleTrackChanged))
        unregisterHandlers.push(realtime.registerHandler('PLAYER_SEEKED', handlePlayerSeeked))
        unregisterHandlers.push(realtime.registerHandler('PLAYER_STATE_UPDATE', handlePlayerStateUpdate))

        // Fetch initial player state with retry
        await fetchPlayerState(3, true)

        // Periodic refresh fallback (client-side polling)
        startPolling()
    }

    /**
     * Reset player state (on disconnect/error)
     */
    function reset() {
        isPlaying.value = false
        lastServerPositionMs.value = 0
        lastServerUpdateAt.value = null
        currentTrack.value = null
        lastEventSeq.value = 0
        lastCommand.value = null
        error.value = null
        sessionId.value = null
        stopPolling()
        clearHandlers()
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
        let completed = false

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const clientMsgId = crypto.randomUUID()
                lastCommand.value = { type: 'pause', at: Date.now() }
                console.info('[player] pause request', {
                    sessionId: sessionId.value,
                    clientMsgId,
                    attempt: attempt + 1
                })
                const response = await apiFetch(`/api/sessions/${sessionId.value}/player/pause`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    credentials: 'include',
                    body: JSON.stringify({ clientMsgId })
                })

                console.debug('[player] pause response', {
                    status: response.status,
                    ok: response.ok
                })

                if (!response.ok) {
                    let errorData: any = null
                    try {
                        errorData = await response.json()
                    } catch (parseError) {
                        console.warn('[player] pause error response not JSON', parseError)
                    }
                    console.error('[player] pause failed', {
                        status: response.status,
                        errorData
                    })
                    const errorCode = errorData?.code || 'PAUSE_FAILED'

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
                completed = true
                return // Success, exit retry loop

            } catch (err: any) {
                if (attempt === maxRetries - 1) {
                    // Final attempt failed
                    error.value = err.message
                    console.error('Failed to pause player after retries:', err)
                }
            } finally {
                if (completed || attempt === maxRetries - 1) {
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
        let completed = false

        for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
                const clientMsgId = crypto.randomUUID()
                lastCommand.value = { type: 'resume', at: Date.now() }
                console.info('[player] resume request', {
                    sessionId: sessionId.value,
                    clientMsgId,
                    attempt: attempt + 1
                })
                const response = await apiFetch(`/api/sessions/${sessionId.value}/player/resume`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    credentials: 'include',
                    body: JSON.stringify({ clientMsgId })
                })

                console.debug('[player] resume response', {
                    status: response.status,
                    ok: response.ok
                })

                if (!response.ok) {
                    let errorData: any = null
                    try {
                        errorData = await response.json()
                    } catch (parseError) {
                        console.warn('[player] resume error response not JSON', parseError)
                    }
                    console.error('[player] resume failed', {
                        status: response.status,
                        errorData
                    })
                    const errorCode = errorData?.code || 'RESUME_FAILED'

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
                completed = true
                return

            } catch (err: any) {
                if (attempt === maxRetries - 1) {
                    error.value = err.message
                    console.error('Failed to resume player after retries:', err)
                }
            } finally {
                if (completed || attempt === maxRetries - 1) {
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
            console.info('[player] next request', {
                sessionId: sessionId.value,
                clientMsgId
            })
            const response = await apiFetch(`/api/sessions/${sessionId.value}/player/next`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ clientMsgId })
            })

            console.debug('[player] next response', {
                status: response.status,
                ok: response.ok
            })

            if (!response.ok) {
                let errorData: any = null
                try {
                    errorData = await response.json()
                } catch (parseError) {
                    console.warn('[player] next error response not JSON', parseError)
                }
                console.error('[player] next failed', {
                    status: response.status,
                    errorData
                })
                throw new Error(errorData?.code || 'NEXT_FAILED')
            }

            const data = await response.json()
            console.log('Next command sent, eventSeq:', data.eventSeq)

            // Proactively refresh in case WS event is missed
            window.setTimeout(() => {
                refreshPlayerState()
            }, 600)

            window.setTimeout(() => {
                refreshPlayerState()
            }, 1800)

        } catch (err: any) {
            error.value = err.message
            console.error('Failed to skip track:', err)
        } finally {
            loading.value.next = false
        }
    }

    /**
     * Seek to position in track
     * Optimistic: updates base position immediately, interpolation continues from there
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

        // Optimistic update: set base position immediately
        // The computed positionMs will automatically reflect this + elapsed time
        lastServerPositionMs.value = targetPositionMs
        lastServerUpdateAt.value = new Date()
        console.info('[player] optimistic seek', {
            targetPositionMs,
            newBaseTime: lastServerUpdateAt.value
        })

        try {
            const clientMsgId = crypto.randomUUID()
            console.info('[player] seek request', {
                sessionId: sessionId.value,
                clientMsgId,
                positionMs: targetPositionMs
            })
            const response = await apiFetch(`/api/sessions/${sessionId.value}/player/seek`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ clientMsgId, positionMs: targetPositionMs })
            })

            console.debug('[player] seek response', {
                status: response.status,
                ok: response.ok
            })

            if (!response.ok) {
                let errorData: any = null
                try {
                    errorData = await response.json()
                } catch (parseError) {
                    console.warn('[player] seek error response not JSON', parseError)
                }
                console.error('[player] seek failed', {
                    status: response.status,
                    errorData
                })
                throw new Error(errorData?.code || 'SEEK_FAILED')
            }

            const data = await response.json()
            console.log('Seek command sent, eventSeq:', data.eventSeq)
            // Server will confirm via WebSocket PLAYER_SEEKED event
            // No need to force refresh - optimistic update is already applied

        } catch (err: any) {
            error.value = err.message
            console.error('Failed to seek:', err)
            // On error, we could revert, but it's better to wait for next server update
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
        lastServerPositionMs.value = payload.positionMs || 0
        lastServerUpdateAt.value = new Date()

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        clearCommandErrorIfMatched()
        console.log('Player paused by', payload.userId)
    }

    function handlePlayerResumed(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = true
        lastServerPositionMs.value = payload.positionMs || 0
        lastServerUpdateAt.value = new Date()

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        clearCommandErrorIfMatched()
        console.log('Player resumed by', payload.userId)
    }

    function handleTrackChanged(message: WSMessage) {
        checkEventSeq(message.eventSeq)

        const payload = message.payload as any
        isPlaying.value = payload.isPlaying ?? true
        lastServerPositionMs.value = payload.positionMs || 0
        lastServerUpdateAt.value = new Date()

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
        lastServerPositionMs.value = payload.positionMs || 0
        lastServerUpdateAt.value = new Date()

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
        const serverPos = payload.positionMs || 0
        const payloadTrackId = payload?.track?.trackId
        const currentTrackId = currentTrack.value?.id
        const trackChanged = Boolean(payloadTrackId && payloadTrackId !== currentTrackId)

        if (trackChanged) {
            lastServerPositionMs.value = serverPos
            lastServerUpdateAt.value = new Date()
        } else {
            const drift = Math.abs(positionMs.value - serverPos)
            if (drift > 2000) {
                lastServerPositionMs.value = serverPos
                lastServerUpdateAt.value = new Date()
            }
        }

        // Update playing state
        isPlaying.value = payload.isPlaying ?? isPlaying.value

        if (payload.track) {
            updateTrack(payload.track)
        }

        // Update generic player state from payload
        updatePlayerState(payload)
        clearCommandErrorIfMatched()
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
            'SPOTIFY_RESTRICTION': 'Playback is restricted on this device or track.',
            'SPOTIFY_FORBIDDEN': 'Spotify refused the playback command.',
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
     * Updates isPlaying and optionally track metadata.
     * Note: base time should only update when base position changes.
     */
    function updatePlayerState(payload: any) {
        isPlaying.value = payload.isPlaying ?? isPlaying.value
        if (payload.track) {
            updateTrack(payload.track)
        }
    }

    function clearCommandErrorIfMatched() {
        if (!lastCommand.value || !error.value) return
        if (error.value === 'SPOTIFY_RESTRICTION') return

        const elapsed = Date.now() - lastCommand.value.at
        if (elapsed > commandClearWindowMs) return

        const expectedIsPlaying = lastCommand.value.type === 'resume'
        if (isPlaying.value === expectedIsPlaying) {
            error.value = null
            lastCommand.value = null
        }
    }

    function clearHandlers() {
        while (unregisterHandlers.length > 0) {
            const unregister = unregisterHandlers.pop()
            try {
                unregister?.()
            } catch (err) {
                console.warn('[player] Failed to unregister handler', err)
            }
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
        // Server-authoritative base values for position computation
        lastServerPositionMs,
        lastServerUpdateAt,
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
        $reset: reset, // Alias for Pinia compatibility
        pausePlayer,
        resumePlayer,
        nextTrack,
        seekTo,
        getFriendlyErrorMessage,
        clearError,
        updatePlayerState,
        updateTrackMetadata,
        refreshPlayerState
    }
})
