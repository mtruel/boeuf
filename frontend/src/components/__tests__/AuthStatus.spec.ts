import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AuthStatus from '../AuthStatus.vue'

global.fetch = vi.fn()

describe('AuthStatus', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        vi.clearAllMocks()
    })

    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('shows loading state initially', () => {
        ; (global.fetch as any).mockImplementation(
            () =>
                new Promise(() => {
                    /* never resolves */
                })
        )

        const wrapper = mount(AuthStatus)
        expect(wrapper.text()).toContain('Vérification de l\'authentification')
    })

    it('displays not authenticated state', async () => {
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({ authenticated: false }),
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        expect(wrapper.text()).toContain('Non connecté')
        expect(wrapper.findComponent({ name: 'LoginButton' }).exists()).toBe(true)
    })

    it('displays authenticated state with user ID', async () => {
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                authenticated: true,
                spotifyUserId: 'test-user-123',
            }),
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        expect(wrapper.text()).toContain('Connecté')
        expect(wrapper.text()).toContain('test-user-123')
        expect(wrapper.findComponent({ name: 'LoginButton' }).exists()).toBe(false)
    })

    it('displays error state on fetch failure', async () => {
        ; (global.fetch as any).mockRejectedValueOnce(new Error('Network error'))

        const wrapper = mount(AuthStatus)
        await flushPromises()

        expect(wrapper.text()).toContain('Erreur')
        expect(wrapper.text()).toContain('Network error')
    })

    it('displays error state on HTTP error', async () => {
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: false,
            status: 500,
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        expect(wrapper.text()).toContain('Erreur')
        expect(wrapper.text()).toContain('HTTP error 500')
    })

    it('allows retrying after error', async () => {
        ; (global.fetch as any).mockRejectedValueOnce(new Error('Network error'))

        const wrapper = mount(AuthStatus)
        await flushPromises()

        expect(wrapper.text()).toContain('Erreur')

            // Mock successful retry
            ; (global.fetch as any).mockResolvedValueOnce({
                ok: true,
                json: async () => ({ authenticated: false }),
            })

        const retryButton = wrapper.find('button')
        await retryButton.trigger('click')
        await flushPromises()

        expect(wrapper.text()).toContain('Non connecté')
    })

    it('has correct data-testid attribute', () => {
        const wrapper = mount(AuthStatus)
        expect(wrapper.find('[data-testid="auth-status"]').exists()).toBe(true)
    })

    it('shows logout button when authenticated', async () => {
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                authenticated: true,
                spotifyUserId: 'test-user-123',
            }),
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        const logoutButton = wrapper.find('[data-testid="logout-button"]')
        expect(logoutButton.exists()).toBe(true)
        expect(logoutButton.text()).toContain('Se déconnecter')
    })

    it('calls logout endpoint and refreshes on logout button click', async () => {
        // Mock initial authenticated state
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                authenticated: true,
                spotifyUserId: 'test-user-123',
            }),
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        // Verify button exists
        const logoutButton = wrapper.find('[data-testid="logout-button"]')
        expect(logoutButton.exists()).toBe(true)

        // Mock logout endpoint
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({ success: true }),
        })

        // Mock refresh after logout
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({ authenticated: false }),
        })

        await logoutButton.trigger('click')
        await flushPromises()

        // Verify logout was called with POST
        expect(global.fetch).toHaveBeenCalledWith('/api/auth/logout', {
            method: 'POST',
        })
    })

    it('handles logout error gracefully', async () => {
        // Mock initial authenticated state
        ; (global.fetch as any).mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                authenticated: true,
                spotifyUserId: 'test-user-123',
            }),
        })

        const wrapper = mount(AuthStatus)
        await flushPromises()

        const logoutButton = wrapper.find('[data-testid="logout-button"]')
        expect(logoutButton.exists()).toBe(true)

        // Mock logout failure
        ; (global.fetch as any).mockRejectedValueOnce(new Error('Logout failed'))

        await logoutButton.trigger('click')
        await flushPromises()

        // Component should still be mounted (error handled gracefully)
        expect(wrapper.exists()).toBe(true)
    })
})
