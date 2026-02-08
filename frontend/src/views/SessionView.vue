<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useRealtimeStore } from '@/stores/realtime'
import { usePresenceStore } from '@/stores/presence'
import { useSessionStore } from '@/stores/session'
import { usePlayerStore } from '@/stores/player'
import SessionAirlock from '@/components/SessionAirlock.vue'
import SessionLive from '@/components/SessionLive.vue'

const route = useRoute()
const realtimeStore = useRealtimeStore()
const presenceStore = usePresenceStore()
const sessionStore = useSessionStore()
const playerStore = usePlayerStore()

const sessionId = computed(() => route.params.sessionId as string)
const currentUserId = ref<string | null>(null)

// Fetch current user ID
async function fetchUserId() {
  try {
    const response = await fetch('/api/auth/status')
    if (response.ok) {
      const data = await response.json()
      currentUserId.value = data.spotifyUserId || null
    }
  } catch (err) {
    console.error('[SessionView] Failed to fetch user ID:', err)
  }
}

onMounted(async () => {
  // Fetch user ID first
  await fetchUserId()
  
  if (!currentUserId.value) {
    console.error('[SessionView] No user ID available')
    return
  }
  
  // Initialize stores
  presenceStore.initialize()
  sessionStore.initialize(sessionId.value, currentUserId.value)
  
  // Load sync state (may restore from sessionStorage)
  // Note: playerStore.init() will be called after successful sync in sessionStore.startListening()
  await sessionStore.loadSyncState()
  
  // Connect WebSocket
  realtimeStore.connect(sessionId.value)
})

onUnmounted(() => {
  realtimeStore.disconnect()
  presenceStore.clear()
  sessionStore.clear()
})
</script>

<template>
  <div>
    <SessionLive v-if="sessionStore.isSynced" />
    <SessionAirlock v-else />
  </div>
</template>
