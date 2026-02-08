import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SessionAirlock from './SessionAirlock.vue'
import { useSessionStore } from '@/stores/session'
import { usePresenceStore } from '@/stores/presence'

describe('SessionAirlock.vue', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        // Mock fetch globally
        global.fetch = vi.fn()
    })

    it('renders airlock component', () => {
        const wrapper = mount(SessionAirlock, {
            global: {
                stubs: {}
            }
        })

        expect(wrapper.find('.airlock-container').exists()).toBe(true)
        expect(wrapper.find('.airlock-main').exists()).toBe(true)
    })

    it('displays session title', () => {
        const wrapper = mount(SessionAirlock)

        const title = wrapper.find('.session-title')
        expect(title.exists()).toBe(true)
        expect(title.text()).toBe('Session d\'écoute')
    })

    it('displays participant count', async () => {
        const wrapper = mount(SessionAirlock)

        // Component renders with initial participant count display
        const count = wrapper.find('.participants-count')
        expect(count.exists()).toBe(true)
        // Will show some count (starts at 0)
        expect(count.text()).toContain('participant')
    })

    it('displays Start Listening button', () => {
        const wrapper = mount(SessionAirlock)

        const button = wrapper.find('.btn-start-listening')
        expect(button.exists()).toBe(true)
        expect(button.text()).toBe('Démarrer l\'écoute')
        expect(button.attributes('disabled')).toBeUndefined()
    })

    it('calls startListening on button click', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        vi.spyOn(sessionStore, 'startListening')

        const button = wrapper.find('.btn-start-listening')
        await button.trigger('click')

        expect(sessionStore.startListening).toHaveBeenCalled()
    })

    it('displays loading spinner when syncing', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Simulate syncing state
        sessionStore.syncState = 'syncing'
        await wrapper.vm.$nextTick()

        const loadingSpinner = wrapper.find('.spinner')
        expect(loadingSpinner.exists()).toBe(true)

        const loadingText = wrapper.find('.loading-content')
        expect(loadingText.text()).toContain('Connexion...')
    })

    it('disables button while syncing', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        sessionStore.syncState = 'syncing'
        await wrapper.vm.$nextTick()

        const button = wrapper.find('.btn-start-listening')
        expect(button.attributes('disabled')).toBeDefined()
    })

    it('displays error message on failure', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Simulate error state
        sessionStore.syncState = 'error'
        sessionStore.error = 'SPOTIFY_NOT_CONNECTED'
        await wrapper.vm.$nextTick()

        const errorMessage = wrapper.find('.error-message')
        expect(errorMessage.exists()).toBe(true)
        expect(errorMessage.text()).toContain('Impossible de synchroniser')
        expect(errorMessage.text()).toContain('Spotify')
    })

    it('displays retry button on error', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        sessionStore.syncState = 'error'
        sessionStore.error = 'SPOTIFY_NOT_CONNECTED'
        await wrapper.vm.$nextTick()

        const retryButton = wrapper.find('.btn-retry')
        expect(retryButton.exists()).toBe(true)
        expect(retryButton.text()).toBe('Réessayer')
    })

    it('calls retry action on retry button click', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        sessionStore.syncState = 'error'
        sessionStore.error = 'SPOTIFY_NOT_CONNECTED'
        await wrapper.vm.$nextTick()

        vi.spyOn(sessionStore, 'retry')

        const retryButton = wrapper.find('.btn-retry')
        await retryButton.trigger('click')

        expect(sessionStore.retry).toHaveBeenCalled()
    })

    it('displays specific error messages for different error codes', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Test SPOTIFY_RATE_LIMITED
        sessionStore.syncState = 'error'
        sessionStore.error = 'SPOTIFY_RATE_LIMITED'
        await wrapper.vm.$nextTick()

        let errorMessage = wrapper.find('.error-message')
        expect(errorMessage.text()).toContain('limite les requêtes')

        // Test SPOTIFY_PLAYER_UNAVAILABLE
        sessionStore.error = 'SPOTIFY_PLAYER_UNAVAILABLE'
        await wrapper.vm.$nextTick()

        errorMessage = wrapper.find('.error-message')
        expect(errorMessage.text()).toContain('appareil Spotify actif')
    })

    it('displays global loading overlay when syncing (AC#4)', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Initially no overlay
        expect(wrapper.find('.global-loading-overlay').exists()).toBe(false)

        // Set syncing state
        sessionStore.syncState = 'syncing'
        await wrapper.vm.$nextTick()

        // Global overlay should appear
        const overlay = wrapper.find('.global-loading-overlay')
        expect(overlay.exists()).toBe(true)
        expect(overlay.attributes('role')).toBe('status')
        expect(overlay.attributes('aria-live')).toBe('polite')

        const spinner = wrapper.find('.large-spinner')
        expect(spinner.exists()).toBe(true)

        const loadingText = wrapper.find('.loading-text')
        expect(loadingText.exists()).toBe(true)
        expect(loadingText.text()).toBe('Connexion en cours...')
    })

    it('hides global loading overlay when not syncing', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Set ready state
        sessionStore.syncState = 'ready'
        await wrapper.vm.$nextTick()

        expect(wrapper.find('.global-loading-overlay').exists()).toBe(false)
    })

    it('displays now playing preview when track available (AC#1)', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        // Set now playing
        sessionStore.nowPlaying = {
            trackId: 'spotify:track:123',
            trackName: 'Test Track',
            artist: 'Test Artist',
            isPlaying: true,
            positionMs: 0,
            durationMs: 180000
        }
        await wrapper.vm.$nextTick()

        const preview = wrapper.find('.now-playing-preview')
        expect(preview.exists()).toBe(true)
        expect(preview.text()).toContain('En cours de lecture')
        expect(preview.text()).toContain('Test Track')
        expect(preview.text()).toContain('Test Artist')
    })

    it('hides now playing preview when no track playing', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        sessionStore.nowPlaying = null
        await wrapper.vm.$nextTick()

        const preview = wrapper.find('.now-playing-preview')
        expect(preview.exists()).toBe(false)
    })

    it('accessibility: error message has role=alert', async () => {
        const wrapper = mount(SessionAirlock)
        const sessionStore = useSessionStore()

        sessionStore.syncState = 'error'
        sessionStore.error = 'SPOTIFY_NOT_CONNECTED'
        await wrapper.vm.$nextTick()

        const errorMessage = wrapper.find('[role="alert"]')
        expect(errorMessage.exists()).toBe(true)
    })
})
