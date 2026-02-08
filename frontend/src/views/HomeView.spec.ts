import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import HomeView from './HomeView.vue'

// Mock fetch globally
global.fetch = vi.fn()

describe('HomeView', () => {
    beforeEach(() => {
        vi.clearAllMocks()
            ; (global.fetch as any).mockResolvedValue({
                ok: true,
                json: async () => ({ status: 'ok' }),
            })
    })

    it('should display Spotify authentication section', () => {
        const wrapper = mount(HomeView)
        expect(wrapper.text()).toContain('Authentification Spotify')
    })

    it('should display backend health status section', () => {
        const wrapper = mount(HomeView)
        expect(wrapper.text()).toContain('Santé du Backend')
    })

    it('should render shadcn-vue Button component', () => {
        const wrapper = mount(HomeView)
        const button = wrapper.find('button')
        expect(button.exists()).toBe(true)
        expect(button.text()).toContain('Actualiser')
    })

    it('should call health check on mount', async () => {
        mount(HomeView)
        await flushPromises()

        // Should call /api/health
        expect(global.fetch).toHaveBeenCalledWith('/api/health')
    })

    it('should render AuthStatus component', () => {
        const wrapper = mount(HomeView)
        expect(wrapper.findComponent({ name: 'AuthStatus' }).exists()).toBe(true)
    })

    it('should display loading state initially', () => {
        const wrapper = mount(HomeView)
        expect(wrapper.text()).toContain('Chargement...')
    })

    it('should display ok status when health check succeeds', async () => {
        (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'ok' }),
        })

        const wrapper = mount(HomeView)
        await flushPromises()

        expect(wrapper.text()).toContain('En ligne')
    })

    it('should display error status when health check fails', async () => {
        // Mock AuthStatus /api/auth/status call (first)
        (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({ authenticated: false }),
        });
        // Mock health check /api/health call (second) - this one fails
        (global.fetch as any).mockResolvedValueOnce({
            ok: false,
        });

        const wrapper = mount(HomeView)
        await flushPromises()

        expect(wrapper.text()).toContain('Hors ligne')
    })
})
