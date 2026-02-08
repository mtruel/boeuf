import './style.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { initApiClient } from './api/client'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// Initialize API client with router (AC 6)
initApiClient(router)

app.mount('#app')
