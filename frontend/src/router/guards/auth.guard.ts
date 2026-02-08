import type { NavigationGuardNext, RouteLocationNormalized } from 'vue-router'
import { useToast } from '@/composables/useToast'

// Cache auth status to avoid repeated API calls (HIGH-5 fix)
let cachedAuthStatus: { authenticated: boolean; timestamp: number } | null = null
const CACHE_TTL = 5000 // 5 seconds

/**
 * Reset auth cache - used for testing
 */
export function resetAuthCache() {
    cachedAuthStatus = null
}

/**
 * Auth Guard - Protects routes requiring authentication (AC 4, 5)
 * 
 * - Blocks access to /session/* routes without auth
 * - Blocks access to /join/* routes without auth
 * - Preserves destination URL in query params for post-login redirect
 * - Optimized with auth status cache to prevent repeated API calls
 */
export async function authGuard(
    to: RouteLocationNormalized,
    from: RouteLocationNormalized,
    next: NavigationGuardNext
) {
    const toast = useToast()

    // Check if route requires auth
    const requiresAuth = to.path.startsWith('/session/') || to.path.startsWith('/join/')

    if (!requiresAuth) {
        next()
        return
    }

    // Check cached auth status first (HIGH-5 optimization)
    const now = Date.now()
    if (cachedAuthStatus && (now - cachedAuthStatus.timestamp) < CACHE_TTL) {
        if (cachedAuthStatus.authenticated) {
            next() // Allow access
            return
        }
        // Fall through to redirect if cached status is unauthenticated
    } else {
        // Cache expired or doesn't exist - check with server
        try {
            const response = await fetch('/api/auth/status', { credentials: 'include' })
            const data = await response.json()

            // Update cache
            cachedAuthStatus = {
                authenticated: data.authenticated,
                timestamp: now
            }

            if (data.authenticated) {
                next() // Allow access
                return
            }
        } catch (error) {
            console.error('Auth guard: Failed to check auth status', error)
            // On error, invalidate cache and assume unauthenticated
            cachedAuthStatus = null
        }
    }

    // User not authenticated - redirect to home
    if (to.path.startsWith('/session/')) {
        // AC 4: Session route protection
        toast.warning('Please sign in to access this session')
        next({
            path: '/',
            query: { redirect: to.fullPath }
        })
    } else if (to.path.startsWith('/join/')) {
        // AC 5: Join route protection
        const inviteCode = to.params.token as string
        toast.warning('Sign in to join this session')
        next({
            path: '/',
            query: { invite: inviteCode }
        })
    } else {
        next()
    }
}
