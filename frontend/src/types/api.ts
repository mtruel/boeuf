/**
 * API Error Response Types (HIGH-8 fix)
 * 
 * Standardized error response structures for type safety
 */

export interface ApiError {
    code: string // Machine-readable error code (e.g., "SPOTIFY_NO_DEVICE")
    message: string // Human-readable error message
    requiresActiveDevice?: boolean // Flag for Spotify device errors (AC 1)
    suggestedAction?: string // Suggested action (e.g., "OPEN_SPOTIFY_WEB_PLAYER")
    metadata?: Record<string, any> // Additional context
}

/**
 * Specific error codes
 */
export const ErrorCodes = {
    // Authentication
    UNAUTHENTICATED: 'UNAUTHENTICATED',
    FORBIDDEN: 'FORBIDDEN',

    // Spotify
    SPOTIFY_NO_DEVICE: 'SPOTIFY_NO_DEVICE',
    SPOTIFY_RATE_LIMITED: 'SPOTIFY_RATE_LIMITED',
    SPOTIFY_UNAVAILABLE: 'SPOTIFY_UNAVAILABLE',
    SPOTIFY_NOT_CONNECTED: 'SPOTIFY_NOT_CONNECTED',
    SPOTIFY_PLAYER_UNAVAILABLE: 'SPOTIFY_PLAYER_UNAVAILABLE',
    SPOTIFY_RESTRICTION: 'SPOTIFY_RESTRICTION',

    // Session
    SESSION_NOT_FOUND: 'SESSION_NOT_FOUND',
    INVALID_REQUEST: 'INVALID_REQUEST',

    // Generic
    INTERNAL_ERROR: 'INTERNAL_ERROR',
    METHOD_NOT_ALLOWED: 'METHOD_NOT_ALLOWED'
} as const

export type ErrorCode = typeof ErrorCodes[keyof typeof ErrorCodes]

/**
 * Type guard to check if error response has requiresActiveDevice flag
 */
export function isDeviceError(error: ApiError): error is ApiError & { requiresActiveDevice: true } {
    return error.code === ErrorCodes.SPOTIFY_NO_DEVICE && error.requiresActiveDevice === true
}

/**
 * Helper to parse error response from fetch
 */
export async function parseApiError(response: Response): Promise<ApiError> {
    try {
        const data = await response.json()
        return {
            code: data.code || 'UNKNOWN_ERROR',
            message: data.message || 'An unknown error occurred',
            requiresActiveDevice: data.requiresActiveDevice,
            suggestedAction: data.suggestedAction,
            metadata: data.metadata
        }
    } catch {
        return {
            code: 'PARSE_ERROR',
            message: `HTTP ${response.status}: ${response.statusText}`
        }
    }
}
