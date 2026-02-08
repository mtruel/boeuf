import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePresenceStore } from '../presence'
import { useRealtimeStore } from '../realtime'
import type { WSMessage } from '../realtime'

let mockWebSocketInstance: any

// Create a mock WebSocket class
class MockWebSocket {
    send = vi.fn()
    close = vi.fn()
    addEventListener = vi.fn()
    removeEventListener = vi.fn()
    readyState = WebSocket.CONNECTING
    onopen: any = null
    onmessage: any = null
    onerror: any = null
    onclose: any = null

    constructor(url: string) {
        mockWebSocketInstance = this
    }
}

describe('Presence Store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        mockWebSocketInstance = null
        global.WebSocket = MockWebSocket as any
    })

    it('should initialize with empty participants', () => {
        const store = usePresenceStore()

        expect(store.participantCount).toBe(0)
        expect(store.onlineCount).toBe(0)
        expect(store.participantList).toEqual([])
    })

    it('should handle SESSION_SNAPSHOT', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        // Initialize presence (registers handlers)
        presenceStore.initialize()

        // Connect realtime (creates WebSocket)
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        // Simulate SESSION_SNAPSHOT message
        const message: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    },
                    {
                        userId: 'user_2',
                        role: 'participant',
                        connectionStatus: 'offline',
                        lastSeenAt: '2026-01-28T11:55:00Z'
                    }
                ]
            }
        }

        // Simulate incoming WebSocket message
        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(message) })

        expect(presenceStore.participantCount).toBe(2)
        expect(presenceStore.onlineCount).toBe(1)
        expect(presenceStore.getParticipant('user_1')).toMatchObject({
            userId: 'user_1',
            role: 'host',
            connectionStatus: 'online'
        })
    })

    it('should handle PARTICIPANT_JOINED', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        presenceStore.initialize()
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        // First, set initial state with snapshot
        const snapshotMessage: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    }
                ]
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(snapshotMessage) })

        expect(presenceStore.participantCount).toBe(1)

        // Now simulate PARTICIPANT_JOINED
        const joinMessage: WSMessage = {
            type: 'PARTICIPANT_JOINED',
            sessionId: 'sess_123',
            eventSeq: 1,
            sentAt: '2026-01-28T12:01:00Z',
            payload: {
                userId: 'user_2',
                role: 'participant',
                timestamp: '2026-01-28T12:01:00Z'
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(joinMessage) })

        expect(presenceStore.participantCount).toBe(2)
        expect(presenceStore.onlineCount).toBe(2)
        expect(presenceStore.getParticipant('user_2')).toMatchObject({
            userId: 'user_2',
            role: 'participant',
            connectionStatus: 'online'
        })
    })

    it('should handle PARTICIPANT_LEFT', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        presenceStore.initialize()
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        // Set initial state
        const snapshotMessage: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    },
                    {
                        userId: 'user_2',
                        role: 'participant',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    }
                ]
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(snapshotMessage) })

        expect(presenceStore.onlineCount).toBe(2)

        // Simulate PARTICIPANT_LEFT
        const leftMessage: WSMessage = {
            type: 'PARTICIPANT_LEFT',
            sessionId: 'sess_123',
            eventSeq: 1,
            sentAt: '2026-01-28T12:05:00Z',
            payload: {
                userId: 'user_2',
                timestamp: '2026-01-28T12:05:00Z'
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(leftMessage) })

        // Participant should still exist but marked offline
        expect(presenceStore.participantCount).toBe(2)
        expect(presenceStore.onlineCount).toBe(1)
        expect(presenceStore.getParticipant('user_2')?.connectionStatus).toBe('offline')
    })

    it('should identify host correctly', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        presenceStore.initialize()
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        const snapshotMessage: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    },
                    {
                        userId: 'user_2',
                        role: 'participant',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    }
                ]
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(snapshotMessage) })


        expect(presenceStore.host).toMatchObject({
            userId: 'user_1',
            role: 'host'
        })
    })

    it('should check if user is online', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        presenceStore.initialize()
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        const snapshotMessage: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    },
                    {
                        userId: 'user_2',
                        role: 'participant',
                        connectionStatus: 'offline',
                        lastSeenAt: '2026-01-28T11:55:00Z'
                    }
                ]
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(snapshotMessage) })


        expect(presenceStore.isOnline('user_1')).toBe(true)
        expect(presenceStore.isOnline('user_2')).toBe(false)
        expect(presenceStore.isOnline('user_unknown')).toBe(false)
    })

    it('should clear all participants', () => {
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        presenceStore.initialize()
        realtimeStore.connect('sess_123')
        mockWebSocketInstance.onopen?.()

        const snapshotMessage: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {
                participants: [
                    {
                        userId: 'user_1',
                        role: 'host',
                        connectionStatus: 'online',
                        lastSeenAt: '2026-01-28T12:00:00Z'
                    }
                ]
            }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(snapshotMessage) })


        expect(presenceStore.participantCount).toBe(1)

        presenceStore.clear()

        expect(presenceStore.participantCount).toBe(0)
        expect(presenceStore.participantList).toEqual([])
    })
})
