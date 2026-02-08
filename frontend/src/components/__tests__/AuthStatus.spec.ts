import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AuthStatus from '../AuthStatus.vue'

global.fetch = vi.fn()

describe('AuthStatus', () => {
    beforeEach(() => {
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
})
