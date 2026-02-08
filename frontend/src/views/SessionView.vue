<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useRealtimeStore } from '@/stores/realtime'
import { usePresenceStore } from '@/stores/presence'
import ParticipantsList from '@/components/ParticipantsList.vue'

const route = useRoute()
const realtimeStore = useRealtimeStore()
const presenceStore = usePresenceStore()

const sessionId = computed(() => route.params.sessionId as string)

onMounted(() => {
  console.log('[SessionView] Mounting, sessionId:', sessionId.value)
  
  // Initialize presence handlers
  presenceStore.initialize()
  
  // Connect WebSocket
  realtimeStore.connect(sessionId.value)
})

onUnmounted(() => {
  console.log('[SessionView] Unmounting, disconnecting WebSocket')
  realtimeStore.disconnect()
  presenceStore.clear()
})
</script>

<template>
  <div class="container mx-auto p-4">
    <h1 class="text-3xl font-bold mb-4">Session {{ sessionId }}</h1>
    
    <!-- Connection Status -->
    <div class="mb-4 p-4 rounded border" :class="{
      'bg-green-100 border-green-300': realtimeStore.isConnected,
      'bg-yellow-100 border-yellow-300': realtimeStore.isConnecting,
      'bg-red-100 border-red-300': realtimeStore.connectionState === 'error',
      'bg-gray-100 border-gray-300': realtimeStore.connectionState === 'disconnected'
    }">
      <div class="flex items-center gap-2">
        <span class="text-sm font-semibold">WebSocket:</span>
        <span class="capitalize">{{ realtimeStore.connectionState }}</span>
        <span v-if="realtimeStore.lastError" class="text-red-600 text-sm ml-2">
          ({{ realtimeStore.lastError }})
        </span>
      </div>
    </div>

    <!-- Participants List -->
    <div class="mb-4">
      <h2 class="text-2xl font-bold mb-2">Participants</h2>
      <ParticipantsList />
    </div>

    <!-- Debug Info -->
    <div class="mt-8 p-4 bg-gray-100 rounded text-xs">
      <h3 class="font-bold mb-2">Debug Info</h3>
      <div><strong>SessionId:</strong> {{ sessionId }}</div>
      <div><strong>Connection State:</strong> {{ realtimeStore.connectionState }}</div>
      <div><strong>Last EventSeq:</strong> {{ realtimeStore.lastEventSeq }}</div>
      <div><strong>Participants Count:</strong> {{ presenceStore.participantCount }}</div>
      <div><strong>Online Count:</strong> {{ presenceStore.onlineCount }}</div>
    </div>
  </div>
</template>
