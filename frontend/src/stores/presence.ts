/**
 * Presence Store - Manages session participants and their connection status
 * 
 * This store tracks who is in the session and whether they're online/offline.
 * It listens to WebSocket events (SESSION_SNAPSHOT, PARTICIPANT_JOINED, PARTICIPANT_LEFT)
 * and updates the participant list accordingly.
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Ref } from 'vue'
import { useRealtimeStore } from './realtime'
import type {
    ParticipantInfo,
    SessionSnapshotPayload,
    ParticipantJoinedPayload,
    ParticipantLeftPayload,
    WSMessage
} from './realtime'

/**
 * Presence Store
 * Manages participant list and presence status
 */
export const usePresenceStore = defineStore('presence', () => {
    // State
    const participants: Ref<Map<string, ParticipantInfo>> = ref(new Map())
    const realtimeStore = useRealtimeStore()

    // Keep handler registrations to avoid duplicates across mounts
    const initialized: Ref<boolean> = ref(false)
    const unregisterHandlers: Array<() => void> = []

    // Computed
    const participantList = computed(() => Array.from(participants.value.values()))
    const participantCount = computed(() => participants.value.size)
    const onlineCount = computed(
        () => participantList.value.filter((p) => p.connectionStatus === 'online').length
    )
    const host = computed(() => participantList.value.find((p) => p.role === 'host'))

    /**
     * Initialize presence tracking for a session
     * Registers message handlers for participant events
     */
    function initialize() {
        if (initialized.value) {
            return
        }

        // Listen for SESSION_SNAPSHOT (initial state)
        unregisterHandlers.push(realtimeStore.onMessage('SESSION_SNAPSHOT', handleSnapshot))

        // Listen for PARTICIPANT_JOINED
        unregisterHandlers.push(realtimeStore.onMessage('PARTICIPANT_JOINED', handleParticipantJoined))

        // Listen for PARTICIPANT_LEFT
        unregisterHandlers.push(realtimeStore.onMessage('PARTICIPANT_LEFT', handleParticipantLeft))

        // Listen for PARTICIPANT_SYNC_STATE_CHANGED
        unregisterHandlers.push(realtimeStore.onMessage('PARTICIPANT_SYNC_STATE_CHANGED', handleSyncStateChanged))

        initialized.value = true
    }

    /**
     * Handle SESSION_SNAPSHOT message (initial state)
     */
    function handleSnapshot(message: WSMessage) {
        const payload = message.payload as SessionSnapshotPayload

        // Replace entire participant map with snapshot
        participants.value.clear()
        payload.participants.forEach((p) => {
            participants.value.set(p.userId, p)
        })
    }

    /**
     * Handle PARTICIPANT_JOINED event
     */
    function handleParticipantJoined(message: WSMessage) {
        const payload = message.payload as ParticipantJoinedPayload

        // Add or update participant
        const existing = participants.value.get(payload.userId)
        if (existing) {
            // Update existing (mark as online)
            existing.connectionStatus = 'online'
            existing.lastSeenAt = payload.timestamp
        } else {
            // Add new participant
            participants.value.set(payload.userId, {
                userId: payload.userId,
                displayName: payload.displayName || payload.userId,
                role: payload.role,
                syncState: 'ready', // Default to ready
                connectionStatus: 'online',
                lastSeenAt: payload.timestamp
            })
        }
    }

    /**
     * Handle PARTICIPANT_LEFT event
     */
    function handleParticipantLeft(message: WSMessage) {
        const payload = message.payload as ParticipantLeftPayload

        // Update participant status to offline
        const participant = participants.value.get(payload.userId)
        if (participant) {
            participant.connectionStatus = 'offline'
            participant.lastSeenAt = payload.timestamp
        }

        // Note: We don't remove the participant from the list
        // They remain in the session but are marked offline
        // Future: Could remove after a timeout or on explicit leave
    }

    /**
     * Handle PARTICIPANT_SYNC_STATE_CHANGED event
     */
    function handleSyncStateChanged(message: WSMessage) {
        const payload = message.payload as { userId: string; syncState: string; timestamp: string }

        // Update participant sync state
        const participant = participants.value.get(payload.userId)
        if (participant) {
            participant.syncState = payload.syncState as 'ready' | 'synced'
            participant.lastSeenAt = payload.timestamp
        }
    }

    /**
     * Get participant by user ID
     */
    function getParticipant(userId: string): ParticipantInfo | undefined {
        return participants.value.get(userId)
    }

    /**
     * Check if user is online
     */
    function isOnline(userId: string): boolean {
        const participant = participants.value.get(userId)
        return participant?.connectionStatus === 'online' || false
    }

    /**
     * Clear all participants (when leaving session)
     */
    function clear() {
        participants.value.clear()

        // Unregister WS handlers so next mount can re-initialize cleanly
        while (unregisterHandlers.length > 0) {
            const unregister = unregisterHandlers.pop()
            try {
                unregister?.()
            } catch {
                // Ignore unregister errors
            }
        }
        initialized.value = false
    }

    return {
        // State
        participants: computed(() => participants.value),

        // Computed
        participantList,
        participantCount,
        onlineCount,
        host,

        // Actions
        initialize,
        getParticipant,
        isOnline,
        clear,
        $reset: clear // Alias for Pinia compatibility
    }
})
