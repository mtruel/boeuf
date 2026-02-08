/**
 * Session Store - Manages session state and sync status
 * 
 * This store handles the participant's sync state (ready/syncing/synced/error)
 * and provides actions to start listening and load sync state.
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Ref } from 'vue'
import { useRealtimeStore } from './realtime'
import type { WSMessage } from './realtime'

export type SyncState = 'ready' | 'syncing' | 'synced' | 'error'

export interface NowPlayingInfo {
    trackId: string
    trackName: string
    artist: string
    isPlaying: boolean
    positionMs: number
    durationMs: number
    imageUrl?: string // Album art URL
}

export interface ParticipantSyncStateChangedPayload {
    userId: string
    syncState: string
    timestamp: string
}

/**
 * Session Store
 * Manages session sync state and now playing information
 */
export const useSessionStore = defineStore('session', () => {
    // State
    const sessionId: Ref<string | null> = ref(null)
    const sessionName: Ref<string | null> = ref(null) // Session title/name
    const syncState: Ref<SyncState> = ref('ready')
    const error: Ref<string | null> = ref(null)
    const nowPlaying: Ref<NowPlayingInfo | null> = ref(null)
    const currentUserId: Ref<string | null> = ref(null)

    const realtimeStore = useRealtimeStore()
    const initialized: Ref<boolean> = ref(false)
    const unregisterHandlers: Array<() => void> = []

    // Computed
    const isReady = computed(() => syncState.value === 'ready')
    const isSyncing = computed(() => syncState.value === 'syncing')
    const isSynced = computed(() => syncState.value === 'synced')
    const hasError = computed(() => syncState.value === 'error')

    /**
     * Initialize session store for a session
     */
    function initialize(sessionIdValue: string, userId: string) {
        if (initialized.value && sessionId.value === sessionIdValue) {
            return
        }

        sessionId.value = sessionIdValue
        currentUserId.value = userId

        // Listen for PARTICIPANT_SYNC_STATE_CHANGED events
        unregisterHandlers.push(
            realtimeStore.onMessage('PARTICIPANT_SYNC_STATE_CHANGED', handleSyncStateChanged)
        )

        initialized.value = true


    }

    /**
     * Handle PARTICIPANT_SYNC_STATE_CHANGED event
     */
    function handleSyncStateChanged(message: WSMessage) {
        const payload = message.payload as ParticipantSyncStateChangedPayload

        // Only update if it's our own sync state change
        if (payload.userId === currentUserId.value) {
            if (payload.syncState === 'synced') {
                syncState.value = 'synced'
            } else if (payload.syncState === 'ready') {
                syncState.value = 'ready'
            }
        }
    }

    /**
     * Load session details from server (name, baseline now playing, etc.)
     */
    async function loadSessionDetails() {
        if (!sessionId.value) {
            console.error('[Session] Cannot load session details: no session ID')
            return
        }

        try {
            const response = await fetch(`/api/sessions/${sessionId.value}`, {
                credentials: 'include'
            })

            if (!response.ok) {
                throw new Error(`Failed to load session details: ${response.status}`)
            }

            const data = await response.json()

            // Use sessionId as name for now (can be enhanced later with user-defined names)
            sessionName.value = data.sessionId

            // Populate nowPlaying from baseline if available
            if (data.nowPlaying) {
                nowPlaying.value = data.nowPlaying
            }

        } catch (err) {
            console.error('[Session] Failed to load session details:', err)
        }
    }

    /**
     * Load sync state from server
     * Called on mount to restore state after page refresh
     */
    async function loadSyncState() {
        if (!sessionId.value) {
            console.error('[Session] Cannot load sync state: no session ID')
            return
        }

        try {
            // Check sessionStorage first for quick restore
            const cached = sessionStorage.getItem(`syncState_${sessionId.value}`)
            if (cached === 'synced') {
                syncState.value = 'synced'
            }

            // Confirm with server
            const response = await fetch(`/api/sessions/${sessionId.value}/me`, {
                credentials: 'include'
            })

            if (!response.ok) {
                throw new Error(`Failed to load sync state: ${response.status}`)
            }

            const data = await response.json()
            syncState.value = data.syncState === 'synced' ? 'synced' : 'ready'

            // Update sessionStorage
            sessionStorage.setItem(`syncState_${sessionId.value}`, data.syncState)

            // If server confirms we're synced, init player store
            if (data.syncState === 'synced') {  // Use server response, not cached state
                const { usePlayerStore } = await import('./player')
                const playerStore = usePlayerStore()
                await playerStore.init(sessionId.value)
            }

        } catch (err) {
            // Silently fail - keep ready state
        }

        // Also load session details to get baseline now playing
        await loadSessionDetails()
    }

    /**
     * Start listening / sync to the session
     */
    async function startListening() {
        if (!sessionId.value) {
            console.error('[Session] Cannot start listening: no session ID')
            return
        }

        syncState.value = 'syncing'
        error.value = null

        try {
            const response = await fetch(`/api/sessions/${sessionId.value}/sync/start`, {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json'
                }
            })

            if (!response.ok) {
                const errorData = await response.json()
                throw new Error(errorData.code || 'SYNC_FAILED')
            }

            const data = await response.json()

            // Update state
            syncState.value = 'synced'
            nowPlaying.value = data.nowPlaying || null

            // Persist to sessionStorage
            sessionStorage.setItem(`syncState_${sessionId.value}`, 'synced')

            // Initialize player store now that we're synced (Story 1.7)
            const { usePlayerStore } = await import('./player')
            const playerStore = usePlayerStore()
            await playerStore.init(sessionId.value)

        } catch (err: any) {
            syncState.value = 'error'
            error.value = err.message || 'Failed to sync'
        }
    }

    /**
     * Retry after error
     */
    function retry() {
        return startListening()
    }

    /**
     * Clear session state (when leaving)
     */
    function clear() {

        if (sessionId.value) {
            sessionStorage.removeItem(`syncState_${sessionId.value}`)
        }

        sessionId.value = null
        sessionName.value = null
        syncState.value = 'ready'
        error.value = null
        nowPlaying.value = null
        currentUserId.value = null

        // Unregister WS handlers
        while (unregisterHandlers.length > 0) {
            const unregister = unregisterHandlers.pop()
            try {
                unregister?.()
            } catch (e) {
                console.warn('[Session] Error unregistering handler:', e)
            }
        }

        initialized.value = false
    }

    return {
        // State
        sessionId,
        sessionName,
        syncState,
        error,
        nowPlaying,

        // Computed
        isReady,
        isSyncing,
        isSynced,
        hasError,

        // Actions
        initialize,
        loadSessionDetails,
        loadSyncState,
        startListening,
        retry,
        clear
    }
})
