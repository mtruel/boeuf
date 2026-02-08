<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { usePlayerStore } from '@/stores/player'
import { usePlayerProgress, formatTime as formatMsToTime } from '@/composables/usePlayerProgress'
import { Play, Pause, SkipForward } from 'lucide-vue-next'

const playerStore = usePlayerStore()
const { currentPositionMs, currentPositionFormatted } = usePlayerProgress()
const { isPlaying, isLoading, hasTrack, track, error } = storeToRefs(playerStore)

// Track when user is dragging the slider
const isDragging = ref(false)
const dragPositionMs = ref(0)

// Computed: Show drag position while dragging, otherwise show actual position
const displayPositionMs = computed(() => {
    return isDragging.value ? dragPositionMs.value : currentPositionMs.value
})

// Computed: Show drag time while dragging
const displayPositionFormatted = computed(() => {
    return isDragging.value ? formatMsToTime(dragPositionMs.value) : currentPositionFormatted.value
})

// Computed: Friendly error message
const friendlyError = computed(() => {
    if (!error.value) return null
    return playerStore.getFriendlyErrorMessage(error.value)
})

// Actions
const togglePlayPause = async () => {
    if (isPlaying.value) {
        await playerStore.pausePlayer()
    } else {
        await playerStore.resumePlayer()
    }
}

const nextTrack = async () => {
    await playerStore.nextTrack()
}

// Handle mousedown - start dragging
const handleMouseDown = () => {
    isDragging.value = true
    dragPositionMs.value = currentPositionMs.value
}

// Handle input during drag - update visual position only
const handleInput = (event: Event) => {
    const target = event.target as HTMLInputElement
    dragPositionMs.value = parseInt(target.value, 10)
}

// Handle change on release - perform the seek
const handleChange = (event: Event) => {
    const target = event.target as HTMLInputElement
    const newPositionMs = parseInt(target.value, 10)
    isDragging.value = false
    // AC 9: Validation is done on backend (reject if positionMs > durationMs)
    playerStore.seekTo(newPositionMs)
}

const dismissError = () => {
    playerStore.clearError()
}

// Format milliseconds to MM:SS
const formatTime = (ms: number): string => {
    return formatMsToTime(ms)
}
</script>

<template>
    <div class="player-controls" :class="{ 'has-error': error }">
        <!-- Error notification -->
        <div v-if="error" class="error-banner" role="alert">
            <p class="error-text">{{ friendlyError }}</p>
            <button @click="dismissError" class="btn-dismiss">Dismiss</button>
        </div>

        <!-- Controls -->
        <div class="controls-row">
            <!-- Play/Pause button -->
            <button
                @click="togglePlayPause"
                :disabled="isLoading || !hasTrack"
                class="btn-control btn-play-pause"
                :class="{ 'btn-loading': isLoading }"
                :title="isPlaying ? 'Pause' : 'Play'"
            >
                <Pause v-if="isPlaying" :size="32" />
                <Play v-else :size="32" />
            </button>

            <!-- Next/Skip button -->
            <button
                @click="nextTrack"
                :disabled="isLoading || !hasTrack"
                class="btn-control btn-skip"
                :class="{ 'btn-loading': isLoading }"
                title="Next track"
            >
                <SkipForward :size="24" />
            </button>
        </div>

        <!-- Position slider (optional MVP) -->
        <div v-if="hasTrack && track" class="progress-container">
            <span class="time-display">{{ displayPositionFormatted }}</span>
            <input
                type="range"
                :min="0"
                :max="track.durationMs"
                :value="displayPositionMs"
                @mousedown="handleMouseDown"
                @input="handleInput"
                @change="handleChange"
                :disabled="isLoading"
                class="progress-slider"
            />
            <span class="time-display">{{ formatTime(track.durationMs) }}</span>
        </div>
    </div>
</template>

<style scoped>
.player-controls {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding: 1.5rem;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    backdrop-filter: blur(10px);
    transition: all 0.3s ease;
}

.player-controls.has-error {
    border: 2px solid #ef4444;
}

/* Error banner */
.error-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    background: rgba(239, 68, 68, 0.1);
    border-left: 4px solid #ef4444;
    border-radius: 6px;
}

.error-text {
    margin: 0;
    color: #ef4444;
    font-size: 0.875rem;
}

.btn-dismiss {
    padding: 0.25rem 0.75rem;
    background: transparent;
    border: 1px solid #ef4444;
    color: #ef4444;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.875rem;
    transition: all 0.2s;
}

.btn-dismiss:hover {
    background: #ef4444;
    color: white;
}

/* Track info */
.track-info {
    display: flex;
    gap: 1rem;
    align-items: center;
}

.track-image {
    width: 64px;
    height: 64px;
    border-radius: 8px;
    overflow: hidden;
    flex-shrink: 0;
}

.track-image img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.track-details {
    flex: 1;
    min-width: 0;
}

.track-name {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: white;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.track-artist {
    margin: 0.25rem 0 0;
    font-size: 0.875rem;
    color: rgba(255, 255, 255, 0.7);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

/* Controls */
.controls-row {
    display: flex;
    gap: 1rem;
    justify-content: center;
    align-items: center;
}

.btn-control {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    transition: all 0.2s ease;
    color: white;
}

.btn-play-pause {
    width: 64px;
    height: 64px;
    background: linear-gradient(135deg, #1db954 0%, #1ed760 100%);
    box-shadow: 0 4px 12px rgba(29, 185, 84, 0.4);
}

.btn-play-pause:hover:not(:disabled) {
    transform: scale(1.05);
    box-shadow: 0 6px 16px rgba(29, 185, 84, 0.6);
}

.btn-skip {
    width: 48px;
    height: 48px;
    background: rgba(255, 255, 255, 0.1);
}

.btn-skip:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.2);
    transform: scale(1.05);
}

.btn-control:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.btn-control.btn-loading {
    opacity: 0.7;
    cursor: wait;
}

/* Progress slider */
.progress-container {
    display: flex;
    align-items: center;
    gap: 0.75rem;
}

.time-display {
    font-size: 0.75rem;
    color: rgba(255, 255, 255, 0.7);
    min-width: 40px;
    text-align: center;
}

.progress-slider {
    flex: 1;
    -webkit-appearance: none;
    appearance: none;
    height: 6px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.2);
    outline: none;
}

.progress-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #1db954;
    cursor: pointer;
    transition: all 0.2s;
}

.progress-slider::-webkit-slider-thumb:hover {
    transform: scale(1.2);
}

.progress-slider::-moz-range-thumb {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #1db954;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
}

.progress-slider::-moz-range-thumb:hover {
    transform: scale(1.2);
}

.progress-slider:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

/* Responsive */
@media (max-width: 640px) {
    .player-controls {
        padding: 1rem;
    }

    .track-image {
        width: 48px;
        height: 48px;
    }

    .btn-play-pause {
        width: 56px;
        height: 56px;
    }

    .btn-skip {
        width: 40px;
        height: 40px;
    }
}
</style>
