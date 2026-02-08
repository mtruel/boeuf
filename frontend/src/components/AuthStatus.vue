<script setup lang="ts">
import { ref, onMounted } from 'vue'
import LoginButton from './LoginButton.vue'

interface AuthStatus {
  authenticated: boolean
  spotifyUserId?: string
}

const authStatus = ref<AuthStatus | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const loggingOut = ref(false)

async function fetchAuthStatus() {
  loading.value = true
  error.value = null

  try {
    const response = await fetch('/api/auth/status')

    if (!response.ok) {
      throw new Error(`HTTP error ${response.status}`)
    }

    authStatus.value = await response.json()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Erreur de connexion'
    if (import.meta.env.DEV) {
      console.error('Failed to fetch auth status:', e)
    }
  } finally {
    loading.value = false
  }
}

async function logout() {
  loggingOut.value = true
  error.value = null

  try {
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
    })

    if (!response.ok) {
      throw new Error(`Logout failed: ${response.status}`)
    }

    // Refresh auth status after successful logout
    await fetchAuthStatus()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Erreur de déconnexion'
    if (import.meta.env.DEV) {
      console.error('Failed to logout:', e)
    }
  } finally {
    loggingOut.value = false
  }
}

onMounted(() => {
  fetchAuthStatus()
})
</script>

<template>
  <div data-testid="auth-status" class="w-full max-w-md">
    <div v-if="loading" class="text-center text-muted-foreground animate-pulse">
      Vérification de l'authentification...
    </div>

    <div v-else-if="error" class="text-center">
      <p class="text-red-500 mb-4">Erreur: {{ error }}</p>
      <button
        @click="fetchAuthStatus"
        class="text-sm text-muted-foreground hover:text-foreground underline"
      >
        Réessayer
      </button>
    </div>

    <div v-else-if="authStatus">
      <!-- Not authenticated state -->
      <div v-if="!authStatus.authenticated" class="text-center space-y-4">
        <div class="bg-muted p-4 rounded-lg">
          <p class="text-muted-foreground mb-2">Non connecté</p>
          <p class="text-sm text-muted-foreground">
            Connecte ton compte Spotify pour commencer à synchroniser la lecture
          </p>
        </div>
        <LoginButton />
      </div>

      <!-- Authenticated state -->
      <div v-else class="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
        <div class="flex items-center gap-3">
          <div class="flex-shrink-0">
            <svg
              class="h-6 w-6 text-green-600 dark:text-green-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          </div>
          <div class="flex-1">
            <p class="font-medium text-green-900 dark:text-green-100">Connecté</p>
            <p class="text-sm text-green-700 dark:text-green-300">
              {{ authStatus.spotifyUserId }}
            </p>
          </div>
          <button
            data-testid="logout-button"
            @click="logout"
            :disabled="loggingOut"
            class="px-3 py-1.5 text-sm font-medium text-red-700 dark:text-red-400 hover:text-red-900 dark:hover:text-red-300 border border-red-300 dark:border-red-700 rounded-md hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loggingOut ? 'Déconnexion...' : 'Se déconnecter' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
