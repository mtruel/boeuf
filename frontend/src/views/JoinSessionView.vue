<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LoginButton from '../components/LoginButton.vue'

interface JoinSessionResponse {
  sessionId: string
}

interface ErrorResponse {
  code: string
  message: string
}

const props = defineProps<{
  token?: string
}>()

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref<string | null>(null)
const sessionId = ref<string | null>(null)
const needsAuth = ref(false)

async function joinSession(inviteToken: string) {
  loading.value = true
  error.value = null

  try {
    const response = await fetch('/api/sessions/join', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ inviteToken })
    })

    if (!response.ok) {
      const errorData: ErrorResponse = await response.json()
      
      if (response.status === 401 || errorData.code === 'UNAUTHENTICATED') {
        needsAuth.value = true
        error.value = 'Connexion Spotify requise pour rejoindre la session'
        return
      }

      error.value = errorData.message || 'Erreur lors de la tentative de rejoindre la session'
      return
    }

    const data: JoinSessionResponse = await response.json()
    sessionId.value = data.sessionId

    // Redirect to session view
    router.push({ name: 'session', params: { sessionId: data.sessionId } })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Erreur de connexion'
    if (import.meta.env.DEV) {
      console.error('Failed to join session:', e)
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const token = props.token || (route.params.token as string)
  
  if (!token) {
    error.value = 'Lien d\'invitation invalide'
    loading.value = false
    return
  }

  joinSession(token)
})
</script>

<template>
  <main class="flex items-center justify-center min-h-[50vh] p-4">
    <div class="w-full max-w-md">
      <h1 class="text-2xl font-bold mb-6 text-center">Rejoindre la session</h1>

      <!-- Loading state -->
      <div v-if="loading" class="text-center">
        <div class="animate-pulse text-muted-foreground">
          Connexion à la session...
        </div>
      </div>

      <!-- Success state -->
      <div v-else-if="sessionId && !error" class="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-6">
        <div class="text-center">
          <svg
            class="h-12 w-12 text-green-600 dark:text-green-400 mx-auto mb-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h2 class="text-xl font-semibold text-green-900 dark:text-green-100 mb-2">
            Session rejointe !
          </h2>
          <p class="text-green-700 dark:text-green-300 mb-4">
            Session ID: <code class="font-mono text-sm">{{ sessionId }}</code>
          </p>
          <p class="text-sm text-muted-foreground">
            Tu peux maintenant écouter avec le groupe
          </p>
        </div>
      </div>

      <!-- Error state - needs auth -->
      <div v-else-if="needsAuth" class="space-y-4">
        <div class="bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <p class="text-yellow-800 dark:text-yellow-200 mb-2">{{ error }}</p>
          <p class="text-sm text-muted-foreground">
            Connecte ton compte Spotify pour continuer
          </p>
        </div>
        <div class="flex justify-center">
          <LoginButton />
        </div>
        <p class="text-xs text-center text-muted-foreground">
          Après connexion, réessaie d'ouvrir le lien d'invitation
        </p>
      </div>

      <!-- Error state - other errors -->
      <div v-else-if="error" class="bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
        <div class="text-center">
          <svg
            class="h-12 w-12 text-red-600 dark:text-red-400 mx-auto mb-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h2 class="text-xl font-semibold text-red-900 dark:text-red-100 mb-2">
            Erreur
          </h2>
          <p class="text-red-700 dark:text-red-300 mb-4">
            {{ error }}
          </p>
          <button
            @click="$router.push('/')"
            class="text-sm text-muted-foreground hover:text-foreground underline"
          >
            Retour à l'accueil
          </button>
        </div>
      </div>
    </div>
  </main>
</template>
