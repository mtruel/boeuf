<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <header class="border-b">
      <div class="container mx-auto px-4 py-4">
        <h1 class="text-2xl font-bold tracking-tight">Boeuf</h1>
      </div>
    </header>

    <RouterView />

    <footer class="border-t mt-8">
      <div class="container mx-auto px-4 py-4 text-center text-xs text-muted-foreground">
        Version 0.0.1 - Frontend Vue 3 + Tailwind v4 + shadcn-vue
      </div>
    </footer>
  </div>
</template>

<style>
/* Reset global styles that might interfere with Tailwind v4 if needed, 
   but @import "tailwindcss" usually handles it. */
</style>
