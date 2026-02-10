<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '@/api/client'

interface DebugParticipant {
  userId: string
  spotifyUserId: string
  displayName: string
  role: string
  joinedAt: string
  lastSeenAt: string
  syncState: string
  connectionStatus: string
}

interface DebugSpotifyState {
  userId: string
  accessible: boolean
  error?: string
  isPlaying: boolean
  trackId?: string
  trackName?: string
  artist?: string
  positionMs: number
  durationMs: number
  imageUrl?: string
}

interface DebugEvent {
  eventSeq: number
  eventType: string
  createdAt: string
  actorUserId?: string
  payload: Record<string, any>
}

interface DebugSessionResponse {
  sessionId: string
  generatedAt: string
  participants: DebugParticipant[]
  spotifyStates: DebugSpotifyState[]
  actionLog: DebugEvent[]
  spotifyChangeLog: DebugEvent[]
}

const route = useRoute()
const sessionId = computed(() => route.params.sessionId as string)

const debugData = ref<DebugSessionResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const lastUpdated = ref<string | null>(null)
let refreshTimer: number | null = null

const spotifyStateByUser = computed<Record<string, DebugSpotifyState>>(() => {
  const map: Record<string, DebugSpotifyState> = {}
  if (!debugData.value) return map
  for (const state of debugData.value.spotifyStates) {
    map[state.userId] = state
  }
  return map
})

async function loadDebug() {
  if (!sessionId.value) return
  loading.value = true
  error.value = null
  try {
    const response = await apiFetch(`/api/sessions/${sessionId.value}/debug`, {
      credentials: 'include'
    })
    if (!response.ok) {
      const errBody = await response.json().catch(() => ({}))
      throw new Error(errBody.code || 'DEBUG_FETCH_FAILED')
    }
    debugData.value = await response.json()
    lastUpdated.value = new Date().toISOString()
  } catch (err: any) {
    error.value = err.message || 'DEBUG_FETCH_FAILED'
  } finally {
    loading.value = false
  }
}

function formatTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('fr-FR')
}

function formatPosition(ms?: number) {
  if (ms === undefined || ms === null) return '-'
  const totalSec = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSec / 60)
  const seconds = totalSec % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

function trackLabel(payload?: Record<string, any>) {
  const track = payload?.track
  if (!track) return '-'
  const name = track.trackName || track.name
  const artist = track.artist
  if (!name && !artist) return '-'
  return [name, artist].filter(Boolean).join(' — ')
}

function spotifyStateLabel(state?: DebugSpotifyState) {
  if (!state) return 'n/a'
  if (!state.accessible) return 'inaccessible'
  return state.isPlaying ? 'playing' : 'paused'
}

onMounted(() => {
  loadDebug()
  refreshTimer = window.setInterval(loadDebug, 5000)
})

onUnmounted(() => {
  if (refreshTimer !== null) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <main class="min-h-screen bg-background text-foreground">
    <div class="container mx-auto px-4 py-8 flex flex-col gap-6">
      <section class="border rounded-xl p-6 bg-card shadow-sm flex flex-col gap-4">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-xs uppercase tracking-wide text-muted-foreground">Debug Session</p>
            <h1 class="text-xl font-semibold">Session {{ sessionId }}</h1>
            <p class="text-sm text-muted-foreground">Derniere mise a jour: {{ formatDateTime(lastUpdated || debugData?.generatedAt) }}</p>
          </div>
          <div class="flex items-center gap-3">
            <span v-if="loading" class="text-sm text-muted-foreground">Chargement...</span>
            <button
              class="px-3 py-2 text-sm rounded-md border bg-background hover:bg-muted transition"
              @click="loadDebug"
            >
              Actualiser
            </button>
          </div>
        </div>
        <p v-if="error" class="text-sm text-destructive">Erreur: {{ error }}</p>
      </section>

      <section class="border rounded-xl p-6 bg-card shadow-sm flex flex-col gap-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Participants</h2>
          <span class="text-sm text-muted-foreground">{{ debugData?.participants.length || 0 }} users</span>
        </div>
        <div class="overflow-auto">
          <table class="min-w-full text-sm">
            <thead class="text-left text-muted-foreground">
              <tr class="border-b">
                <th class="py-2 pr-4">Nom</th>
                <th class="py-2 pr-4">Spotify ID</th>
                <th class="py-2 pr-4">Role</th>
                <th class="py-2 pr-4">Sync</th>
                <th class="py-2 pr-4">Connexion</th>
                <th class="py-2 pr-4">Spotify</th>
                <th class="py-2 pr-4">Track</th>
                <th class="py-2 pr-4">Position</th>
                <th class="py-2 pr-4">Rejoint</th>
                <th class="py-2 pr-4">Derniere activite</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="participant in debugData?.participants" :key="participant.userId" class="border-b">
                <td class="py-2 pr-4 font-medium">{{ participant.displayName }}</td>
                <td class="py-2 pr-4 font-mono text-xs">{{ participant.spotifyUserId }}</td>
                <td class="py-2 pr-4">{{ participant.role }}</td>
                <td class="py-2 pr-4">{{ participant.syncState }}</td>
                <td class="py-2 pr-4">
                  <span
                    class="inline-flex items-center rounded-full px-2 py-1 text-xs"
                    :class="participant.connectionStatus === 'online' ? 'bg-green-100 text-green-700' : 'bg-muted text-muted-foreground'"
                  >
                    {{ participant.connectionStatus }}
                  </span>
                </td>
                <td class="py-2 pr-4">
                  <span
                    class="inline-flex items-center rounded-full px-2 py-1 text-xs"
                    :class="spotifyStateByUser[participant.userId]?.accessible ? 'bg-blue-100 text-blue-700' : 'bg-muted text-muted-foreground'"
                  >
                    {{ spotifyStateLabel(spotifyStateByUser[participant.userId]) }}
                  </span>
                </td>
                <td class="py-2 pr-4">
                  {{ spotifyStateByUser[participant.userId]?.trackName || '-' }}
                </td>
                <td class="py-2 pr-4">
                  {{ formatPosition(spotifyStateByUser[participant.userId]?.positionMs) }}
                </td>
                <td class="py-2 pr-4">{{ formatDateTime(participant.joinedAt) }}</td>
                <td class="py-2 pr-4">{{ formatDateTime(participant.lastSeenAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="grid lg:grid-cols-2 gap-6">
        <div class="border rounded-xl p-6 bg-card shadow-sm flex flex-col gap-4">
          <h2 class="text-lg font-semibold">Actions boeuf detectees</h2>
          <div v-if="!debugData?.actionLog.length" class="text-sm text-muted-foreground">
            Aucun event.
          </div>
          <ul v-else class="flex flex-col gap-3">
            <li v-for="event in debugData?.actionLog" :key="event.eventSeq" class="border rounded-lg p-3">
              <div class="flex items-center justify-between">
                <span class="text-xs text-muted-foreground">{{ formatTime(event.createdAt) }}</span>
                <span class="text-xs font-mono">#{{ event.eventSeq }}</span>
              </div>
              <div class="mt-1 flex items-center justify-between">
                <span class="font-medium">{{ event.eventType }}</span>
                <span class="text-xs text-muted-foreground">{{ event.actorUserId || '-' }}</span>
              </div>
              <div class="mt-1 text-sm text-muted-foreground">
                {{ trackLabel(event.payload) }}
              </div>
            </li>
          </ul>
        </div>

        <div class="border rounded-xl p-6 bg-card shadow-sm flex flex-col gap-4">
          <h2 class="text-lg font-semibold">Changements Spotify detectes</h2>
          <div v-if="!debugData?.spotifyChangeLog.length" class="text-sm text-muted-foreground">
            Aucun event.
          </div>
          <ul v-else class="flex flex-col gap-3">
            <li v-for="event in debugData?.spotifyChangeLog" :key="event.eventSeq" class="border rounded-lg p-3">
              <div class="flex items-center justify-between">
                <span class="text-xs text-muted-foreground">{{ formatTime(event.createdAt) }}</span>
                <span class="text-xs font-mono">#{{ event.eventSeq }}</span>
              </div>
              <div class="mt-1 flex items-center justify-between">
                <span class="font-medium">{{ event.eventType }}</span>
                <span class="text-xs text-muted-foreground">{{ event.payload?.polledUserId || '-' }}</span>
              </div>
              <div class="mt-1 text-sm text-muted-foreground">
                {{ trackLabel(event.payload) }}
              </div>
            </li>
          </ul>
        </div>
      </section>
    </div>
  </main>
</template>
