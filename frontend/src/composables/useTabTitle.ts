/**
 * useTabTitle - Composable to manage browser tab title and favicon based on player state
 *
 * Handles:
 * - Dynamic tab title based on playback state
 * - Format: "▶ {trackName} - {artist}" when playing
 * - Format: "⏸ Paused" when paused
 * - Format: "boeuf - {sessionName}" when idle
 * - Favicon badge for sync status
 *
 * AC 7: Feedback tab en arrière-plan
 */

import { watch, onUnmounted } from 'vue'
import { usePlayerStore } from '@/stores/player'
import { useSessionStore } from '@/stores/session'

export function useTabTitle() {
    const playerStore = usePlayerStore()
    const sessionStore = useSessionStore()

    // Store original title and favicon for restoration
    const originalTitle = typeof document !== 'undefined' ? document.title : 'boeuf'

    // Create or get favicon element
    const getFaviconLink = (): HTMLLinkElement => {
        let link = document.querySelector('link[rel="icon"]') as HTMLLinkElement
        if (!link) {
            link = document.createElement('link')
            link.rel = 'icon'
            document.head.appendChild(link)
        }
        return link
    }

    /**
     * Update favicon with badge based on sync status
     * Generates a dynamic favicon with a green badge when synced
     */
    const updateFavicon = (synced: boolean) => {
        try {
            const link = getFaviconLink()

            if (!synced) {
                // No badge - use default favicon
                link.href = '/favicon.ico'
                return
            }

            // Generate favicon with green badge using canvas
            const canvas = document.createElement('canvas')
            canvas.width = 32
            canvas.height = 32
            const ctx = canvas.getContext('2d')

            if (!ctx) return

            // Draw base circle (vinyl record style)
            ctx.fillStyle = '#d97706' // Orange-600 (brand color)
            ctx.beginPath()
            ctx.arc(16, 16, 14, 0, Math.PI * 2)
            ctx.fill()

            // Draw center hole (vinyl style)
            ctx.fillStyle = '#1a1816' // Charcoal
            ctx.beginPath()
            ctx.arc(16, 16, 6, 0, Math.PI * 2)
            ctx.fill()

            // Draw green badge (top-right)
            ctx.fillStyle = '#22c55e' // Green-500 (synced indicator)
            ctx.beginPath()
            ctx.arc(26, 6, 6, 0, Math.PI * 2)
            ctx.fill()

            // Convert canvas to data URL
            link.href = canvas.toDataURL('image/png')
        } catch (e) {
            // Favicon update can fail in some contexts; silently ignore
            console.debug('[useTabTitle] Failed to update favicon:', e)
        }
    }

    /**
     * Update document title based on player state
     */
    const updateTitle = () => {
        const track = playerStore.currentTrack
        const isPlaying = playerStore.isPlaying

        if (!track || !track.name) {
            // No track or idle state
            const sessionName = sessionStore.sessionName || 'Session'
            document.title = `boeuf - ${sessionName}`
            updateFavicon(false)
            return
        }

        if (isPlaying) {
            // Playing state: "▶ {trackName} - {artist}"
            const artist = track.artist || 'Unknown Artist'
            document.title = `▶ ${track.name} - ${artist}`
            updateFavicon(true)
        } else {
            // Paused state: "⏸ Paused"
            document.title = `⏸ Paused`
            updateFavicon(false)
        }
    }

    /**
     * Watch player state and update tab title
     */
    watch(
        () => playerStore.isPlaying,
        () => updateTitle()
    )

    watch(
        () => playerStore.currentTrack?.id,
        () => updateTitle()
    )

    // Initial update
    updateTitle()

    /**
     * Cleanup: restore original title on unmount
     */
    onUnmounted(() => {
        if (typeof document !== 'undefined') {
            document.title = originalTitle
        }
    })

    return {
        updateFavicon
    }
}
