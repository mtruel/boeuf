<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import AuthStatus from '@/components/AuthStatus.vue'
import CreateSessionComponent from '@/components/CreateSessionComponent.vue'
import TheWelcome from '../components/TheWelcome.vue'

const route = useRoute()
const router = useRouter()
const healthStatus = ref<'loading' | 'ok' | 'error'>('loading')
const backendData = ref<any>(null)
const isAuthenticated = ref(false)

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

async function checkAuthStatus() {
  try {
    const response = await fetch('/api/auth/status')
    if (response.ok) {
      const data = await response.json()
      const wasUnauthenticated = !isAuthenticated.value
      isAuthenticated.value = data.authenticated
      
      // Handle post-login redirects (AC 4, 5)
      if (wasUnauthenticated && data.authenticated) {
        handlePostLoginRedirect()
      }
    } else {
      isAuthenticated.value = false
    }
  } catch (e) {
    isAuthenticated.value = false
  }
}

function handleAuthChanged(authenticated: boolean) {
  isAuthenticated.value = authenticated
}

/**
 * Handle post-login redirects from query params (AC 4, 5)
 */
function handlePostLoginRedirect() {
  const { redirect, invite } = route.query
  
  // AC 4: Redirect to preserved session URL
  if (redirect && typeof redirect === 'string') {
    router.push(redirect)
    return
  }
  
  // AC 5: Auto-join with preserved invite code
  if (invite && typeof invite === 'string') {
    router.push({ name: 'join-session', params: { token: invite } })
    return
  }
}

onMounted(() => {
  checkHealth()
  checkAuthStatus()
})

// Re-check auth status when AuthStatus component emits updates
// Simple polling for now (can be improved with event bus later)
setInterval(checkAuthStatus, 2000)
</script>

<template>
  <main class="container mx-auto px-4 py-8 flex flex-col items-center gap-6">
    <div class="w-full max-w-md bg-card border rounded-xl p-6 shadow-sm flex flex-col gap-6">
      <!-- Spotify Authentication Section -->
      <section class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">Authentification Spotify</h2>
        <AuthStatus @auth-changed="handleAuthChanged" />
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

      <!-- Session Creation Section (AC 3: Only show when authenticated) -->
      <section v-if="isAuthenticated" class="flex flex-col gap-3">
        <h2 class="text-lg font-semibold">Session</h2>
        <CreateSessionComponent />
      </section>
    </div>
  </main>
</template>
