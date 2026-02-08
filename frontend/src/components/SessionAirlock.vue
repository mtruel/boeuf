<script setup lang="ts">
import { computed } from 'vue'
import { useSessionStore } from '@/stores/session'
import { usePresenceStore } from '@/stores/presence'

const sessionStore = useSessionStore()
const presenceStore = usePresenceStore()

// Computed states
const isSyncing = computed(() => sessionStore.isSyncing)
const hasError = computed(() => sessionStore.hasError)
const errorMessage = computed(() => sessionStore.error)
const displayErrorMessage = computed(() => {
    const code = errorMessage.value
    if (!code) return null

    switch (code) {
        case 'SPOTIFY_NOT_CONNECTED':
            return 'Impossible de synchroniser. Veuillez connecter votre compte Spotify, puis réessayer.'
        case 'SPOTIFY_RATE_LIMITED':
            return 'Spotify limite les requêtes. Veuillez patienter un moment puis réessayer.'
        case 'SPOTIFY_PLAYER_UNAVAILABLE':
            return 'Aucun appareil Spotify actif trouvé. Démarrez Spotify sur n\'importe quel appareil, puis réessayez.'
        case 'SPOTIFY_UNAVAILABLE':
            return 'Spotify est temporairement indisponible. Veuillez réessayer dans un moment.'
        default:
            return 'Impossible de synchroniser. Veuillez vérifier votre connexion Spotify et réessayer.'
    }
})

// Actions
const handleStartListening = () => {
    sessionStore.startListening()
}

const handleRetry = () => {
    sessionStore.retryWithBackoff()
}
</script>

<template>
    <div class="airlock-container">
        <!-- Global Loading Overlay (AC#4 feedback intermédiaire global) -->
        <div v-if="isSyncing" class="global-loading-overlay" role="status" aria-live="polite">
            <div class="loading-spinner-container">
                <div class="large-spinner"></div>
                <p class="loading-text">Connexion en cours...</p>
            </div>
        </div>

        <div class="airlock-content">
            <!-- Background album art (blurred/grayscale) -->
            <div class="airlock-background">
                <!-- Future: show album art if available -->
                <div class="gradient-overlay"></div>
            </div>

            <!-- Main content -->
            <div class="airlock-main">
                <!-- Session title (AC#1) -->
                <h1 class="session-title" :title="`Session ID: ${sessionStore.sessionId}`">
                    {{ sessionStore.sessionName || 'Session d\'écoute' }}
                </h1>
                
                <!-- Now playing info if available (AC#1 context) -->
                <div v-if="sessionStore.nowPlaying" class="now-playing-preview">
                    <p class="preview-label">En cours de lecture</p>
                    <p class="preview-track">{{ sessionStore.nowPlaying.trackName }}</p>
                    <p class="preview-artist">{{ sessionStore.nowPlaying.artist }}</p>
                </div>

                <!-- Participants present -->
                <div class="participants-info">
                    <p class="participants-count">
                        {{ presenceStore.participantCount }}
                        {{ presenceStore.participantCount === 1 ? 'participant' : 'participants' }}
                    </p>
                </div>

                <!-- CTA Button -->
                <div class="cta-container">
                    <button
                        v-if="!hasError"
                        @click="handleStartListening"
                        :disabled="isSyncing"
                        class="btn-start-listening"
                        :class="{ 'btn-loading': isSyncing }"
                    >
                        <span v-if="!isSyncing">Démarrer l'écoute</span>
                        <span v-else class="loading-content">
                            <span class="spinner"></span>
                            Connexion...
                        </span>
                    </button>

                    <!-- Error state -->
                    <div v-if="hasError" class="error-container">
                        <p class="error-message" role="alert">{{ displayErrorMessage }}</p>
                        <button @click="handleRetry" class="btn-retry">Réessayer</button>
                    </div>
                </div>

                <p class="airlock-subtitle">Cliquez pour rejoindre la session d'écoute</p>
            </div>
        </div>
    </div>
</template>

<style scoped>
.airlock-container {
    position: fixed;
    inset: 0;
    background: #1a1816; /* Charcoal */
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
}

.airlock-content {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.airlock-background {
    position: absolute;
    inset: 0;
    filter: blur(50px) grayscale(100%);
    opacity: 0.3;
}

.gradient-overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, rgba(26, 24, 22, 0.8) 0%, rgba(26, 24, 22, 0.95) 100%);
}

.airlock-main {
    position: relative;
    z-index: 1;
    text-align: center;
    max-width: 600px;
    padding: 2rem;
}

.session-title {
    font-family: 'Fraunces', serif;
    font-size: 3.5rem;
    font-weight: 700;
    color: #f5f5f4; /* Stone-100 */
    margin-bottom: 2rem;
    line-height: 1.2;
}

.participants-info {
    margin-bottom: 3rem;
}

.participants-count {
    font-size: 1.25rem;
    color: #a8a29e; /* Stone-400 */
    font-weight: 500;
}

.now-playing-preview {
    margin-bottom: 2.5rem;
    padding: 1rem;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 0.5rem;
    border-left: 3px solid #d97706; /* Amber accent */
}

.preview-label {
    font-size: 0.875rem;
    color: #78716c; /* Stone-500 */
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.5rem;
    font-weight: 600;
}

.preview-track {
    font-size: 1.125rem;
    font-weight: 600;
    color: #f5f5f4; /* Stone-100 */
    margin-bottom: 0.25rem;
}

.preview-artist {
    font-size: 1rem;
    color: #a8a29e; /* Stone-400 */
}

.cta-container {
    margin-bottom: 1.5rem;
}

.btn-start-listening {
    background: #d97706; /* Amber-600 */
    color: #fafaf9; /* Stone-50 */
    font-size: 1.5rem;
    font-weight: 600;
    padding: 1.25rem 3rem;
    border: none;
    border-radius: 0.75rem;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(217, 119, 6, 0.3);
    min-width: 240px;
}

.btn-start-listening:hover:not(:disabled) {
    background: #f59e0b; /* Amber-500 */
    box-shadow: 0 6px 16px rgba(217, 119, 6, 0.4);
    transform: translateY(-2px);
}

.btn-start-listening:active:not(:disabled) {
    transform: translateY(0);
    box-shadow: 0 2px 8px rgba(217, 119, 6, 0.3);
}

.btn-start-listening:disabled {
    opacity: 0.7;
    cursor: not-allowed;
}

.btn-start-listening.btn-loading {
    background: #ea580c; /* Orange-600 */
}

.loading-content {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
}

.spinner {
    display: inline-block;
    width: 20px;
    height: 20px;
    border: 3px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: #fff;
    animation: spin 0.8s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

.airlock-subtitle {
    font-size: 1rem;
    color: #78716c; /* Stone-500 */
    font-weight: 400;
}

.error-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
}

.error-message {
    color: #ef4444; /* Red-500 */
    font-size: 1.125rem;
    font-weight: 500;
    background: rgba(239, 68, 68, 0.1);
    padding: 1rem 1.5rem;
    border-radius: 0.5rem;
    border: 1px solid rgba(239, 68, 68, 0.3);
}

.btn-retry {
    background: #d97706; /* Amber-600 */
    color: #fafaf9;
    font-size: 1.125rem;
    font-weight: 600;
    padding: 0.75rem 2rem;
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    transition: all 0.2s ease;
}

.btn-retry:hover {
    background: #f59e0b;
}

/* Global Loading Overlay (AC#4) */
.global-loading-overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(26, 24, 22, 0.92); /* Semi-transparent Charcoal */
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(8px);
}

.loading-spinner-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.5rem;
}

.large-spinner {
    width: 64px;
    height: 64px;
    border: 6px solid rgba(217, 119, 6, 0.2); /* Amber-600 semi-transparent */
    border-radius: 50%;
    border-top-color: #d97706; /* Amber-600 solid */
    animation: spin-large 1s linear infinite;
}

@keyframes spin-large {
    to {
        transform: rotate(360deg);
    }
}

.loading-text {
    font-size: 1.25rem;
    font-weight: 600;
    color: #f5f5f4; /* Stone-100 */
    letter-spacing: 0.02em;
}

/* Accessibility: Reduced Motion */
@media (prefers-reduced-motion: reduce) {
    .btn-start-listening:hover:not(:disabled) {
        transform: none;
    }

    .spinner {
        animation: none;
        border-top-color: rgba(255, 255, 255, 0.3);
    }

    .large-spinner {
        animation: none;
        border-top-color: rgba(217, 119, 6, 0.2);
    }

    .global-loading-overlay {
        backdrop-filter: none;
    }
}

/* Focus styles for accessibility */
.btn-start-listening:focus-visible,
.btn-retry:focus-visible {
    outline: 2px solid #e5e7eb; /* Gray-200 */
    outline-offset: 4px;
}
</style>
