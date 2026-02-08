<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { usePresenceStore } from '@/stores/presence'
import { usePlayerStore } from '@/stores/player'
import { useTabTitle } from '@/composables/useTabTitle'
import PlayerControls from './PlayerControls.vue'

const sessionStore = useSessionStore()
const presenceStore = usePresenceStore()
const playerStore = usePlayerStore()

// AC 7: Initialize tab title management
useTabTitle()

// AC 3: Track change transition animation state
const albumArtKey = ref(0)

// Computed - Use playerStore.track as source of truth for current playback
const nowPlaying = computed(() => playerStore.track)
const participantList = computed(() => presenceStore.participantList)
const hasNowPlaying = computed(() => nowPlaying.value !== null)
const isPlayerPaused = computed(() => !playerStore.isPlaying && hasNowPlaying.value)

/**
 * Format artist names for display
 */
const artistNames = computed(() => {
    if (!nowPlaying.value?.artist) {
        return ''
    }
    return nowPlaying.value.artist
})

/**
 * Get album art image - uses Spotify API image URL from backend
 */
const albumArtUrl = computed(() => {
    // Use imageUrl from playerStore track if available
    if (nowPlaying.value?.imageUrl) {
        return nowPlaying.value.imageUrl
    }
    return null
})

/**
 * Fallback gradient when no track image available
 */
const albumGradient = computed(() => {
    if (albumArtUrl.value) {
        return `url('${albumArtUrl.value}')`
    }
    return 'linear-gradient(135deg, #d97706 0%, #f59e0b 100%)'
})

// Watch for track changes to trigger animation
const trackId = computed(() => nowPlaying.value?.id)
watch(trackId, () => {
    // AC 3: Trigger cross-fade animation by incrementing key
    // This causes Vue to re-mount the image element, triggering CSS transitions
    albumArtKey.value++
})
</script>

<template>
    <div class="live-container">
        <!-- Sync Status Badge -->
        <div class="status-bar">
            <div class="status-badge status-synced">
                <span class="status-dot"></span>
                <span class="status-text">Live</span>
            </div>
        </div>

        <!-- Now Playing Section -->
        <div class="now-playing-section" v-if="hasNowPlaying">
            <div class="album-art" :class="{ 'paused': isPlayerPaused }">
                <!-- Album art with fallback (AC#2, AC#3 cross-fade transition, AC#4 paused state) -->
                <img 
                    v-if="albumArtUrl" 
                    :key="`${albumArtKey}-${albumArtUrl}`"
                    :src="albumArtUrl" 
                    :alt="`${nowPlaying?.trackName} album art`"
                    class="album-image"
                    :class="{ 'paused': isPlayerPaused }"
                    @error="() => {}"
                />
                <div v-else class="album-placeholder">♪</div>
            </div>

            <div class="track-info">
                <h2 class="track-name">{{ nowPlaying?.name || 'No track playing' }}</h2>
                <p class="track-artist">{{ artistNames }}</p>
            </div>

            <!-- Player Controls (Story 1.7, AC 8, AC 9) -->
            <PlayerControls />
        </div>

        <div class="empty-state" v-else>
            <p class="empty-message">No track playing</p>
            <p class="empty-hint">Start playing music on Spotify to see it here</p>
        </div>

        <!-- Participants List -->
        <div class="participants-section">
            <h3 class="section-title">
                Participants ({{ participantList.length }})
            </h3>

            <div class="participants-list">
                <div
                    v-for="participant in participantList"
                    :key="participant.userId"
                    class="participant-item"
                    :class="{
                        'participant-online': participant.connectionStatus === 'online',
                        'participant-offline': participant.connectionStatus === 'offline'
                    }"
                >
                    <div class="participant-avatar">
                        {{ participant.displayName.charAt(0).toUpperCase() }}
                    </div>

                    <div class="participant-info">
                        <p class="participant-name">
                            {{ participant.displayName }}
                            <span v-if="participant.role === 'host'" class="host-badge">Host</span>
                        </p>
                        <p class="participant-status">
                            <span class="status-indicator" :class="`status-${participant.syncState}`"></span>
                            {{ participant.syncState === 'synced' ? 'Synced' : 'Ready' }}
                        </p>
                    </div>

                    <div class="connection-indicator" :class="`connection-${participant.connectionStatus}`">
                        {{ participant.connectionStatus === 'online' ? '●' : '○' }}
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.live-container {
    min-height: 100vh;
    background: #1a1816; /* Charcoal */
    color: #f5f5f4; /* Stone-100 */
    padding: 2rem;
    animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: scale(0.98);
    }
    to {
        opacity: 1;
        transform: scale(1);
    }
}

/* Reduced Motion */
@media (prefers-reduced-motion: reduce) {
    .live-container {
        animation: none;
    }

    .album-art {
        transition: none;
    }

    .album-image {
        animation: none;
        transition: none;
    }

    .status-dot {
        animation: none;
    }
}

.status-bar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 2rem;
}

.status-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    border-radius: 9999px;
    font-size: 0.875rem;
    font-weight: 600;
}

.status-synced {
    background: rgba(34, 197, 94, 0.1); /* Green-500 with opacity */
    color: #22c55e; /* Green-500 */
}

.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
    animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
}

.now-playing-section {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 1rem;
    padding: 2rem;
    margin-bottom: 2rem;
}

.album-art {
    width: 200px;
    height: 200px;
    margin: 0 auto 1.5rem;
    border-radius: 0.75rem;
    overflow: hidden;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    background: linear-gradient(135deg, #d97706 0%, #f59e0b 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    transition: filter 0.3s ease-out;
}

/* AC 4: Paused state - dim the container */
.album-art.paused {
    filter: grayscale(50%);
}

.album-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
    /* AC 3: Cross-fade transition for track changes (300ms) */
    animation: albumFadeIn 0.3s ease-out;
    opacity: 1;
    transition: opacity 0.3s ease-out;
}

@keyframes albumFadeIn {
    from {
        opacity: 0;
    }
    to {
        opacity: 1;
    }
}

/* AC 4: Paused state visual (dimmed album image) */
.album-image.paused {
    opacity: 0.7;
}

.album-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #d97706 0%, #f59e0b 100%);
    font-size: 4rem;
    color: #fafaf9;
}

.track-info {
    text-align: center;
    margin-bottom: 1.5rem;
}

.track-name {
    font-family: 'Fraunces', serif;
    font-size: 2rem;
    font-weight: 700;
    color: #f5f5f4;
    margin-bottom: 0.5rem;
}

.track-artist {
    font-size: 1.25rem;
    color: #a8a29e; /* Stone-400 */
}

.playback-controls {
    text-align: center;
}

.playback-status {
    font-size: 1rem;
    color: #78716c; /* Stone-500 */
    font-weight: 500;
}

.empty-state {
    text-align: center;
    padding: 4rem 2rem;
}

.empty-message {
    font-size: 1.5rem;
    color: #a8a29e;
    font-weight: 600;
    margin-bottom: 0.5rem;
}

.empty-hint {
    font-size: 1rem;
    color: #78716c;
}

.participants-section {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 1rem;
    padding: 1.5rem;
}

.section-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: #f5f5f4;
    margin-bottom: 1rem;
}

.participants-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.participant-item {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem;
    border-radius: 0.5rem;
    background: rgba(255, 255, 255, 0.03);
    transition: background 0.2s ease;
}

.participant-item:hover {
    background: rgba(255, 255, 255, 0.06);
}

.participant-offline {
    opacity: 0.6;
}

.participant-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: linear-gradient(135deg, #d97706 0%, #f59e0b 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    color: #fafaf9;
    font-size: 1.125rem;
}

.participant-info {
    flex: 1;
}

.participant-name {
    font-size: 1rem;
    font-weight: 600;
    color: #f5f5f4;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.host-badge {
    font-size: 0.75rem;
    font-weight: 600;
    background: rgba(217, 119, 6, 0.2);
    color: #d97706; /* Amber-600 */
    padding: 0.125rem 0.5rem;
    border-radius: 0.25rem;
}

.participant-status {
    font-size: 0.875rem;
    color: #a8a29e;
    display: flex;
    align-items: center;
    gap: 0.375rem;
    margin-top: 0.25rem;
}

.status-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
}

.status-synced {
    background: #22c55e; /* Green-500 */
}

.status-ready {
    background: #a8a29e; /* Stone-400 */
}

.connection-indicator {
    font-size: 1.25rem;
    line-height: 1;
}

.connection-online {
    color: #22c55e; /* Green-500 */
}

.connection-offline {
    color: #78716c; /* Stone-500 */
}
</style>
