/**
 * Realtime Store - Manages WebSocket connection and message dispatching
 * 
 * This store handles the WebSocket lifecycle (connect, disconnect, reconnect)
 * and dispatches incoming messages to appropriate domain stores.
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Ref } from 'vue'

// WebSocket message types (must match backend UPPER_SNAKE convention)
export type MessageType =
    | 'SESSION_SNAPSHOT'
    | 'PARTICIPANT_JOINED'
    | 'PARTICIPANT_LEFT'
    | 'PARTICIPANT_SYNC_STATE_CHANGED'
    | 'PLAYER_PAUSED'
    | 'PLAYER_RESUMED'
    | 'TRACK_CHANGED'
    | 'QUEUE_UPDATED'
    | 'HOST_CHANGED'
    | 'WS_ERROR'
    | 'WS_FORBIDDEN'

// WebSocket connection states
export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

// WebSocket message envelope (matches backend Message struct)
export interface WSMessage {
    type: MessageType
    sessionId: string
    eventSeq: number
    sentAt: string // RFC3339 timestamp
    payload: unknown
}

// Session snapshot payload (initial state on connection)
export interface SessionSnapshotPayload {
    participants: ParticipantInfo[]
    nowPlaying?: NowPlayingInfo | null
}

export interface ParticipantInfo {
    userId: string
    role: 'host' | 'participant'
    syncState: 'ready' | 'synced'
    lastSeenAt: string // RFC3339
    connectionStatus: 'online' | 'offline'
}

export interface NowPlayingInfo {
    trackId: string
    trackName: string
    artist: string
    isPlaying: boolean
    positionMs: number
}

// Participant joined event payload
export interface ParticipantJoinedPayload {
    userId: string
    role: 'host' | 'participant'
    timestamp: string // RFC3339
}

// Participant left event payload
export interface ParticipantLeftPayload {
    userId: string
    timestamp: string // RFC3339
}

// Error payload
export interface ErrorPayload {
    code: string
    message: string
    details?: string
}

/**
 * Realtime Store
 * Manages WebSocket connection and dispatches messages to domain stores
 */
export const useRealtimeStore = defineStore('realtime', () => {
    // State
    const ws: Ref<WebSocket | null> = ref(null)
    const connectionState: Ref<ConnectionState> = ref('disconnected')
    const sessionId: Ref<string | null> = ref(null)
    const lastError: Ref<string | null> = ref(null)
    const lastEventSeq: Ref<number> = ref(-1)

    // Message handlers registry (other stores can register handlers)
    const messageHandlers: Map<MessageType, Array<(message: WSMessage) => void>> = new Map()

    // Computed
    const isConnected = computed(() => connectionState.value === 'connected')
    const isConnecting = computed(() => connectionState.value === 'connecting')

    /**
     * Connect to WebSocket for a session
     * @param sid Session ID to connect to
     */
    function connect(sid: string) {
        if (ws.value) {
            disconnect()
        }

        sessionId.value = sid
        connectionState.value = 'connecting'
        lastError.value = null

        // Determine WebSocket URL (protocol depends on window.location)
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/ws/${sid}`

        // Log only in development
        if (import.meta.env.DEV) {
            console.log('[Realtime] Connecting to', wsUrl)
        }

        try {
            const socket = new WebSocket(wsUrl)

            socket.onopen = () => {
                connectionState.value = 'connected'
                lastError.value = null
            }

            socket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data) as WSMessage
                    handleMessage(message)
                } catch (error) {
                    console.error('[Realtime] Failed to parse message:', error)
                }
            }

            socket.onerror = (event) => {
                console.error('[Realtime] WebSocket error:', event)
                connectionState.value = 'error'
                lastError.value = 'WebSocket connection error'
            }

            socket.onclose = (event) => {
                connectionState.value = 'disconnected'
                ws.value = null

                // Future: automatic reconnection logic (Story 4.2)
                if (!event.wasClean) {
                    lastError.value = `Connection closed unexpectedly: ${event.code}`
                }
            }

            ws.value = socket
        } catch (error) {
            console.error('[Realtime] Failed to create WebSocket:', error)
            connectionState.value = 'error'
            lastError.value = error instanceof Error ? error.message : 'Unknown error'
        }
    }

    /**
     * Disconnect from WebSocket
     */
    function disconnect() {
        if (ws.value) {
            ws.value.close(1000, 'Client initiated disconnect')
            ws.value = null
        }
        connectionState.value = 'disconnected'
        sessionId.value = null
        lastEventSeq.value = -1
    }

    /**
     * Handle incoming WebSocket message
     * @param message Parsed WebSocket message
     */
    function handleMessage(message: WSMessage) {
        // Check for out-of-order messages
        if (lastEventSeq.value >= 0 && message.eventSeq <= lastEventSeq.value) {
            console.warn(`[Realtime] Received out-of-order message: eventSeq=${message.eventSeq}, expected >${lastEventSeq.value}`)
        }

        // Update last event sequence
        if (message.eventSeq > lastEventSeq.value) {
            lastEventSeq.value = message.eventSeq
        }

        // Dispatch to registered handlers
        const handlers = messageHandlers.get(message.type)
        if (handlers && handlers.length > 0) {
            handlers.forEach((handler) => {
                try {
                    handler(message)
                } catch (error) {
                    console.error(`[Realtime] Handler error for ${message.type}:`, error)
                }
            })
        }
    }

    /**
     * Register a message handler
     * @param type Message type to handle
     * @param handler Handler function
     */
    function onMessage(type: MessageType, handler: (message: WSMessage) => void) {
        if (!messageHandlers.has(type)) {
            messageHandlers.set(type, [])
        }
        messageHandlers.get(type)!.push(handler)

        // Return unregister function
        return () => {
            const handlers = messageHandlers.get(type)
            if (handlers) {
                const index = handlers.indexOf(handler)
                if (index !== -1) {
                    handlers.splice(index, 1)
                }
            }
        }
    }

    /**
     * Send a message (for future client->server messages)
     * MVP: Not used yet, but prepared for future stories
     */
    function send(message: unknown) {
        if (ws.value && ws.value.readyState === WebSocket.OPEN) {
            ws.value.send(JSON.stringify(message))
        }
    }

    return {
        // State
        connectionState,
        sessionId,
        lastError,
        lastEventSeq,

        // Computed
        isConnected,
        isConnecting,

        // Actions
        connect,
        disconnect,
        onMessage,
        send
    }
})
