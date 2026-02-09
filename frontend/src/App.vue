<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toaster } from 'vue-sonner'

const isLocalhost = typeof window !== 'undefined' && window.location.hostname === 'localhost'
const configuredPublicUrl = (import.meta.env.VITE_PUBLIC_URL as string | undefined) || 'http://127.0.0.1:3000'
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <header class="border-b">
      <div class="container mx-auto px-4 py-4">
        <RouterLink to="/" class="inline-block">
          <h1 class="text-2xl font-bold tracking-tight hover:text-primary transition-colors cursor-pointer">Boeuf</h1>
        </RouterLink>
        <div
          v-if="isLocalhost"
          class="mt-3 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900"
        >
          Attention: l'app est ouverte via localhost. Utilise
          <a :href="configuredPublicUrl" class="font-medium underline hover:text-amber-700">
            {{ configuredPublicUrl }}
          </a>
          pour eviter les erreurs OAuth Spotify.
        </div>
      </div>
    </header>

    <RouterView />

    <footer class="border-t mt-8">
      <div class="container mx-auto px-4 py-4 text-center text-xs text-muted-foreground">
        Version 0.0.1 - Frontend Vue 3 + Tailwind v4 + shadcn-vue
      </div>
    </footer>
    
    <Toaster position="top-right" richColors />
  </div>
</template>

<style>
/* Reset global styles that might interfere with Tailwind v4 if needed, 
   but @import "tailwindcss" usually handles it. */
</style>
