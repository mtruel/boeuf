/**
 * API Client with 401 Interceptor (AC 6, 7)
 * 
 * Wraps native fetch to intercept 401 errors and handle session expiration
 */

import { useToast } from '@/composables/useToast'
import type { Router } from 'vue-router'
import { resetAuthCache } from '@/router/guards/auth.guard'

let sessionExpiredToastShown = false
let sessionExpiredToastTimer: ReturnType<typeof setTimeout> | null = null
let router: Router | null = null

/**
 * Initialize the API client with router instance
 */
export function initApiClient(routerInstance: Router) {
    router = routerInstance
}

/**
 * Clear all stores on 401 (AC 6)
 */
async function clearAllStores() {
    try {
        const { useSessionStore } = await import('@/stores/session')
        const { usePlayerStore } = await import('@/stores/player')
        const { usePresenceStore } = await import('@/stores/presence')
        const { useRealtimeStore } = await import('@/stores/realtime')

        const sessionStore = useSessionStore()
        const playerStore = usePlayerStore()
        const presenceStore = usePresenceStore()
        const realtimeStore = useRealtimeStore()

        sessionStore.clear()
        playerStore.$reset()
        presenceStore.$reset()
        realtimeStore.disconnect()
    } catch (error) {
        console.error('Failed to clear stores:', error)
    }
}

/**
 * Enhanced fetch with 401 interceptor (AC 6, 7)
 */
export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    const response = await fetch(input, init)

    // Intercept 401 errors (AC 6)
    if (response.status === 401) {
        const toast = useToast()

    // Clear all stores (fire-and-forget to avoid race condition)
    clearAllStores().catch(console.error)
    resetAuthCache()

        // Show toast once (debounce to avoid spam - AC 7)
        if (!sessionExpiredToastShown) {
            sessionExpiredToastShown = true
            toast.error('Your session expired. Please sign in again.')

            // Clear existing timer to prevent memory leak
            if (sessionExpiredToastTimer) {
                clearTimeout(sessionExpiredToastTimer)
            }

            // Reset flag after 5s
            sessionExpiredToastTimer = setTimeout(() => {
                sessionExpiredToastShown = false
                sessionExpiredToastTimer = null
            }, 5000)
        }

        // Redirect to home
        if (router) {
            router.push('/')
        }
    }

    return response
}

/**
 * Reset session expired flag (for testing)
 */
export function resetSessionExpiredFlag() {
    sessionExpiredToastShown = false
}
