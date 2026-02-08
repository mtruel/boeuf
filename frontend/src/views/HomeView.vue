<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import AuthStatus from '@/components/AuthStatus.vue'
import CreateSessionComponent from '@/components/CreateSessionComponent.vue'
import TheWelcome from '../components/TheWelcome.vue'

const healthStatus = ref<'loading' | 'ok' | 'error'>('loading')
const backendData = ref<any>(null)

async function checkHealth() {
  healthStatus.value = 'loading'
  try {
    const response = await fetch('/api/health')
    if (response.ok) {
      backendData.value = await response.json()
      healthStatus.value = 'ok'
    } else {
      healthStatus.value = 'error'
    }
  } catch (e) {
    healthStatus.value = 'error'
    console.error('Failed to fetch health:', e)
  }
}

onMounted(() => {
  checkHealth()
})
</script>

<template>
  <main class="container mx-auto px-4 py-8 flex flex-col items-center gap-6">
    <div class="w-full max-w-md bg-card border rounded-xl p-6 shadow-sm flex flex-col gap-6">
      <!-- Spotify Authentication Section -->
      <section class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">Authentification Spotify</h2>
        <AuthStatus />
      </section>

      <!-- Backend Health Check Section -->
      <section class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">Santé du Backend</h2>
        <div class="flex items-center justify-between">
          <span class="font-medium">État:</span>
          <div class="flex items-center gap-2">
            <span v-if="healthStatus === 'loading'" class="text-yellow-500 animate-pulse">Chargement...</span>
            <span v-else-if="healthStatus === 'ok'" class="text-green-500 font-bold">En ligne</span>
            <span v-else class="text-red-500 font-bold">Hors ligne</span>
          </div>
        </div>

        <div v-if="backendData" class="bg-muted p-3 rounded text-xs font-mono">
          <pre>{{ JSON.stringify(backendData, null, 2) }}</pre>
        </div>

        <div class="flex justify-center">
          <Button @click="checkHealth" variant="outline">
            Actualiser
          </Button>
        </div>
      </section>

      <!-- Session Creation Section -->
      <section class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">Session</h2>
        <CreateSessionComponent />
      </section>
    </div>
  </main>
</template>
