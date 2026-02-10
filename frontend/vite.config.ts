import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    vueDevTools(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/ws': {
        target: 'ws://backend-dev:8080',
        ws: true,
        changeOrigin: true,
      },
      '/api': {
        target: 'http://backend-dev:8080',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://backend-dev:8080',
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
})
