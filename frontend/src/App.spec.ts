import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import App from './App.vue'

// Mock fetch globally
global.fetch = vi.fn()

describe('App', () => {
    beforeEach(() => {
        vi.clearAllMocks()
            ; (global.fetch as any).mockResolvedValue({
                ok: true,
                json: async () => ({ status: 'ok' }),
            })
    })

    it('should mount and render the app title', () => {
        const wrapper = mount(App)
        expect(wrapper.find('h1').text()).toContain('Boeuf')
    })

    it('should display Spotify authentication section', () => {
        const wrapper = mount(App)
        expect(wrapper.text()).toContain('Authentification Spotify')
    })

    it('should display backend health status section', () => {
        const wrapper = mount(App)
        expect(wrapper.text()).toContain('Santé du Backend')
    })

    it('should render shadcn-vue Button component', () => {
        const wrapper = mount(App)
        const button = wrapper.find('button')
        expect(button.exists()).toBe(true)
        expect(button.text()).toContain('Actualiser')
    })

    it('should call health check and auth status on mount', async () => {
        mount(App)
        await flushPromises()

        // Should call both /api/health and /api/auth/status
        expect(global.fetch).toHaveBeenCalledWith('/api/health')
        expect(global.fetch).toHaveBeenCalledWith('/api/auth/status')
    })

    it('should render AuthStatus component', () => {
        const wrapper = mount(App)
        expect(wrapper.findComponent({ name: 'AuthStatus' }).exists()).toBe(true)
    })
})

