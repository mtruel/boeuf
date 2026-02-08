import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import LoginButton from '../LoginButton.vue'

describe('LoginButton', () => {
    it('renders correctly', () => {
        const wrapper = mount(LoginButton)
        expect(wrapper.find('button').exists()).toBe(true)
        expect(wrapper.text()).toContain('Connecter Spotify')
    })

    it('has correct button styling', () => {
        const wrapper = mount(LoginButton)
        const button = wrapper.find('button')
        expect(button.classes()).toContain('bg-green-600')
    })

    it('displays Spotify icon', () => {
        const wrapper = mount(LoginButton)
        expect(wrapper.find('svg').exists()).toBe(true)
    })

    it('redirects to /auth/spotify/start on click', async () => {
        // Mock window.location.href
        delete (window as any).location
        window.location = { href: '' } as any

        const wrapper = mount(LoginButton)
        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start')
    })
})
