<template>
  <div class="create-session-component">
    <!-- Create Session Button -->
    <button 
      v-if="!session"
      @click="createSession" 
      :disabled="isLoading"
      data-testid="create-session-btn"
      class="bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white font-semibold py-2 px-4 rounded transition-colors"
    >
      {{ isLoading ? 'Création...' : 'Créer une session' }}
    </button>

    <!-- Error Message -->
    <div 
      v-if="error" 
      data-testid="error-message"
      class="mt-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded"
    >
      {{ error }}
    </div>

    <!-- Session Created - Show Invite Info -->
    <div v-if="session" class="mt-6 space-y-4">
      <h3 class="text-lg font-semibold text-green-600">🎉 Session créée !</h3>
      
      <div class="bg-gray-50 p-4 rounded-lg">
        <label class="block text-sm font-medium text-gray-700 mb-2">
          Lien d'invitation
        </label>
        <div class="flex items-center space-x-2">
          <input 
            :value="session.inviteUrl"
            readonly
            data-testid="invite-url"
            class="flex-1 p-2 border border-gray-300 rounded bg-white text-sm"
          />
          <button 
            @click="copyToClipboard"
            data-testid="copy-btn"
            class="bg-green-600 hover:bg-green-700 text-white px-3 py-2 rounded text-sm transition-colors"
          >
            📋 Copier
          </button>
        </div>
      </div>

      <!-- Copy Feedback -->
      <div 
        v-if="copySuccess"
        data-testid="copy-feedback"
        class="text-green-600 text-sm"
      >
        ✅ Copié!
      </div>

      <!-- Session Info -->
      <div class="text-sm text-gray-600">
        <p><strong>ID Session:</strong> {{ session.sessionId }}</p>
        <p data-testid="expiration-date"><strong>Expire le:</strong> {{ formatExpirationDate(session.expiresAt) }}</p>
      </div>

      <!-- Action Buttons -->
      <div class="flex gap-2">
        <button 
          @click="goToSession"
          class="bg-green-600 hover:bg-green-700 text-white py-2 px-4 rounded transition-colors"
        >
          🎵 Ouvrir la session
        </button>
        <button 
          @click="resetComponent"
          class="bg-gray-600 hover:bg-gray-700 text-white py-2 px-4 rounded transition-colors"
        >
          Créer une nouvelle session
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiFetch } from '@/api/client'

interface SessionResponse {
  sessionId: string
  inviteUrl: string
  inviteCode: string
  expiresAt: string
}

interface ErrorResponse {
  code: string
  message: string
}

const router = useRouter()
const isLoading = ref(false)
const error = ref<string>('')
const session = ref<SessionResponse | null>(null)
const copySuccess = ref(false)

const createSession = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    const response = await apiFetch('/api/sessions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Pour inclure les cookies de session
      body: JSON.stringify({})
    })

    if (!response.ok) {
      const errorData: ErrorResponse = await response.json()
      throw new Error(errorData.message || 'Erreur lors de la création de la session')
    }

    const sessionData: SessionResponse = await response.json()
    session.value = sessionData
    
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Une erreur est survenue'
    console.error('Create session error:', err)
  } finally {
    isLoading.value = false
  }
}

const copyToClipboard = async () => {
  if (!session.value) return

  try {
    await navigator.clipboard.writeText(session.value.inviteUrl)
    copySuccess.value = true
    
    // Reset feedback after 2 seconds
    setTimeout(() => {
      copySuccess.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
    // Fallback for older browsers
    const textArea = document.createElement('textarea')
    textArea.value = session.value.inviteUrl
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    copySuccess.value = true
    setTimeout(() => {
      copySuccess.value = false
    }, 2000)
  }
}

const formatExpirationDate = (isoDate: string): string => {
  const date = new Date(isoDate)
  
  // Check if date is invalid
  if (isNaN(date.getTime())) {
    console.error('Invalid date format:', isoDate)
    return 'Date invalide'
  }
  
  return date.toLocaleString('fr-FR', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const goToSession = () => {
  if (!session.value) return
  router.push({ name: 'session', params: { sessionId: session.value.sessionId } })
}

const resetComponent = () => {
  session.value = null
  error.value = ''
  copySuccess.value = false
}
</script>

<style scoped>
/* Additional component-specific styles if needed */
</style>