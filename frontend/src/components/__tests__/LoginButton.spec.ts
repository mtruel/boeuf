import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import LoginButton from '../LoginButton.vue'

// Mock window.location.href
delete (window as any).location
window.location = { href: '' } as any

describe('LoginButton', () => {
    const router = createRouter({
        history: createMemoryHistory(),
        routes: [{ path: '/', component: { template: '<div>Home</div>' } }]
    })

    it('renders correctly', async () => {
        await router.push('/')
        const wrapper = mount(LoginButton, {
            global: { plugins: [router] }
        })
        expect(wrapper.find('button').exists()).toBe(true)
        expect(wrapper.text()).toContain('Connecter Spotify')
    })

    it('has correct button styling', async () => {
        await router.push('/')
        const wrapper = mount(LoginButton, {
            global: { plugins: [router] }
        })
        const button = wrapper.find('button')
        expect(button.classes()).toContain('bg-green-600')
    })

    it('displays Spotify icon', async () => {
        await router.push('/')
        const wrapper = mount(LoginButton, {
            global: { plugins: [router] }
        })
        expect(wrapper.find('svg').exists()).toBe(true)
    })

    it('redirects to /auth/spotify/start on click', async () => {
        await router.push('/')
        window.location.href = ''

        const wrapper = mount(LoginButton, {
            global: { plugins: [router] }
        })
        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start')
    })
})
