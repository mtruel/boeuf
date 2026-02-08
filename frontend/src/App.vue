<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'

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
  <div class="min-h-screen bg-background text-foreground p-8 flex flex-col items-center justify-center gap-6">
    <header class="text-center">
      <h1 class="text-4xl font-bold tracking-tight">Boeuf 🥩</h1>
      <p class="text-muted-foreground mt-2">Squelette exécutable frontend/backend</p>
    </header>

    <main class="w-full max-w-md bg-card border rounded-xl p-6 shadow-sm flex flex-col gap-4">
      <div class="flex items-center justify-between">
        <span class="font-medium">État du Backend:</span>
        <div class="flex items-center gap-2">
          <span v-if="healthStatus === 'loading'" class="text-yellow-500 animate-pulse">Chargement...</span>
          <span v-else-if="healthStatus === 'ok'" class="text-green-500 font-bold">En ligne</span>
          <span v-else class="text-red-500 font-bold">Hors ligne</span>
        </div>
      </div>

      <div v-if="backendData" class="bg-muted p-3 rounded text-xs font-mono">
        <pre>{{ JSON.stringify(backendData, null, 2) }}</pre>
      </div>

      <div class="pt-4 flex justify-center">
        <Button @click="checkHealth" variant="outline">
          Actualiser
        </Button>
      </div>
    </main>

    <footer class="text-xs text-muted-foreground">
      Version 0.0.1 - Frontend Vue 3 + Tailwind v4 + shadcn-vue
    </footer>
  </div>
</template>

<style>
/* Reset global styles that might interfere with Tailwind v4 if needed, 
   but @import "tailwindcss" usually handles it. */
</style>
