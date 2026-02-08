import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createPinia } from 'pinia'
import App from './App.vue'
import HomeView from './views/HomeView.vue'

describe('App', () => {
    it('should mount and render the app structure with router', async () => {
        const router = createRouter({
            history: createMemoryHistory(),
            routes: [
                { path: '/', component: HomeView },
            ],
        })

        const pinia = createPinia()

        const wrapper = mount(App, {
            global: {
                plugins: [router, pinia],
                stubs: {
                    RouterLink: false // Don't stub RouterLink so h1 renders
                }
            },
        })

        await router.isReady()
        await wrapper.vm.$nextTick()

        // Verify app structure
        expect(wrapper.find('header').exists()).toBe(true)
        expect(wrapper.html()).toContain('Boeuf')
        expect(wrapper.find('footer').exists()).toBe(true)
        expect(wrapper.find('footer').text()).toContain('Version 0.0.1')
    })

    it('should render RouterView component', () => {
        const router = createRouter({
            history: createMemoryHistory(),
            routes: [
                { path: '/', component: HomeView },
            ],
        })

        const pinia = createPinia()

        const wrapper = mount(App, {
            global: {
                plugins: [router, pinia],
            },
        })

        // RouterView should be present
        expect(wrapper.findComponent({ name: 'RouterView' }).exists()).toBe(true)
    })
})

