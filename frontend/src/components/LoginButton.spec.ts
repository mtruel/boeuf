import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import LoginButton from './LoginButton.vue'

// Mock window.location.href
delete (window as any).location
window.location = { href: '' } as any

describe('LoginButton - returnTo functionality', () => {
    let router: any

    beforeEach(() => {
        router = createRouter({
            history: createMemoryHistory(),
            routes: [
                { path: '/', name: 'home', component: { template: '<div>Home</div>' } },
                { path: '/join/:token', name: 'join', component: { template: '<div>Join</div>' } }
            ]
        })
        window.location.href = ''
    })

    it('should include return_to query param when returnTo prop is provided', async () => {
        await router.push('/')

        const wrapper = mount(LoginButton, {
            props: {
                returnTo: '/join/abc123'
            },
            global: {
                plugins: [router]
            }
        })

        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start?return_to=%2Fjoin%2Fabc123')
    })

    it('should use current route as return_to when no prop provided', async () => {
        await router.push('/join/xyz789')

        const wrapper = mount(LoginButton, {
            global: {
                plugins: [router]
            }
        })

        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start?return_to=%2Fjoin%2Fxyz789')
    })

    it('should not include return_to when on home route', async () => {
        await router.push('/')

        const wrapper = mount(LoginButton, {
            global: {
                plugins: [router]
            }
        })

        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start')
    })

    it('should prioritize returnTo prop over current route', async () => {
        await router.push('/some/other/path')

        const wrapper = mount(LoginButton, {
            props: {
                returnTo: '/join/priority'
            },
            global: {
                plugins: [router]
            }
        })

        await wrapper.find('button').trigger('click')

        expect(window.location.href).toBe('/auth/spotify/start?return_to=%2Fjoin%2Fpriority')
    })
})
