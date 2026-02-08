import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
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

describe('Realtime Store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        mockWebSocketInstance = null
        global.WebSocket = MockWebSocket as any
    })

    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('should initialize with disconnected state', () => {
        const store = useRealtimeStore()

        expect(store.connectionState).toBe('disconnected')
        expect(store.isConnected).toBe(false)
        expect(store.isConnecting).toBe(false)
        expect(store.sessionId).toBeNull()
    })

    it('should connect to WebSocket', () => {
        const store = useRealtimeStore()
        const sessionId = 'sess_test_123'

        store.connect(sessionId)

        expect(store.connectionState).toBe('connecting')
        expect(store.sessionId).toBe(sessionId)
        // WebSocket instance should have been created
        expect(mockWebSocketInstance).toBeTruthy()
    })

    it('should transition to connected state on open', () => {
        const store = useRealtimeStore()
        store.connect('sess_test_123')

        // Simulate WebSocket open
        mockWebSocketInstance.onopen?.()

        expect(store.connectionState).toBe('connected')
        expect(store.isConnected).toBe(true)
    })

    it('should handle incoming messages', () => {
        const store = useRealtimeStore()
        const handler = vi.fn()

        store.onMessage('SESSION_SNAPSHOT', handler)
        store.connect('sess_test_123')

        // Simulate incoming message
        const message: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_test_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: { participants: [] }
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(message) })

        expect(handler).toHaveBeenCalledWith(message)
        expect(store.lastEventSeq).toBe(0)
    })

    it('should track event sequence', () => {
        const store = useRealtimeStore()
        store.connect('sess_test_123')

        // Simulate messages with increasing eventSeq
        const msg1: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_test_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {}
        }

        const msg2: WSMessage = {
            type: 'PARTICIPANT_JOINED',
            sessionId: 'sess_test_123',
            eventSeq: 1,
            sentAt: '2026-01-28T12:00:01Z',
            payload: {}
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(msg1) })
        expect(store.lastEventSeq).toBe(0)

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(msg2) })
        expect(store.lastEventSeq).toBe(1)
    })

    it('should disconnect cleanly', () => {
        const store = useRealtimeStore()
        store.connect('sess_test_123')
        mockWebSocketInstance.onopen?.()

        store.disconnect()

        expect(mockWebSocketInstance.close).toHaveBeenCalledWith(1000, 'Client initiated disconnect')
        expect(store.connectionState).toBe('disconnected')
        expect(store.sessionId).toBeNull()
    })

    it('should handle connection errors', () => {
        const store = useRealtimeStore()
        store.connect('sess_test_123')

        mockWebSocketInstance.onerror?.({})

        expect(store.connectionState).toBe('error')
        expect(store.lastError).toBeTruthy()
    })

    it('should handle unexpected disconnection', () => {
        const store = useRealtimeStore()
        store.connect('sess_test_123')
        mockWebSocketInstance.onopen?.()

        // Simulate unexpected close
        mockWebSocketInstance.onclose?.({ wasClean: false, code: 1006, reason: 'Connection lost' })

        expect(store.connectionState).toBe('disconnected')
        expect(store.lastError).toBeTruthy()
    })

    it('should allow multiple handlers for same message type', () => {
        const store = useRealtimeStore()
        const handler1 = vi.fn()
        const handler2 = vi.fn()

        store.onMessage('SESSION_SNAPSHOT', handler1)
        store.onMessage('SESSION_SNAPSHOT', handler2)
        store.connect('sess_test_123')

        const message: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_test_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {}
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(message) })

        expect(handler1).toHaveBeenCalledOnce()
        expect(handler2).toHaveBeenCalledOnce()
    })

    it('should allow unregistering handlers', () => {
        const store = useRealtimeStore()
        const handler = vi.fn()

        const unregister = store.onMessage('SESSION_SNAPSHOT', handler)
        store.connect('sess_test_123')

        // Unregister handler
        unregister()

        const message: WSMessage = {
            type: 'SESSION_SNAPSHOT',
            sessionId: 'sess_test_123',
            eventSeq: 0,
            sentAt: '2026-01-28T12:00:00Z',
            payload: {}
        }

        mockWebSocketInstance.onmessage?.({ data: JSON.stringify(message) })

        expect(handler).not.toHaveBeenCalled()
    })

    it('should warn on out-of-order messages', () => {
        const store = useRealtimeStore()
        const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => { })
        store.connect('sess_test_123')

        // Send message with seq 5 first
        mockWebSocketInstance.onmessage?.({ data: JSON.stringify({ type: 'SESSION_SNAPSHOT', sessionId: 'sess_test_123', eventSeq: 5, sentAt: '2026-01-28T12:00:00Z', payload: {} }) })
        expect(store.lastEventSeq).toBe(5)

        // Then send message with seq 3 (out of order)
        mockWebSocketInstance.onmessage?.({ data: JSON.stringify({ type: 'PARTICIPANT_JOINED', sessionId: 'sess_test_123', eventSeq: 3, sentAt: '2026-01-28T12:00:01Z', payload: {} }) })

        expect(consoleWarnSpy).toHaveBeenCalledWith(expect.stringContaining('out-of-order'))
        consoleWarnSpy.mockRestore()
    })
})
