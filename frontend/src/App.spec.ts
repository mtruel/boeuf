import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
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

        const wrapper = mount(App, {
            global: {
                plugins: [router],
            },
        })

        await router.isReady()

        // Verify app structure
        expect(wrapper.find('header').exists()).toBe(true)
        expect(wrapper.find('h1').text()).toBe('Boeuf')
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

        const wrapper = mount(App, {
            global: {
                plugins: [router],
            },
        })

        // RouterView should be present
        expect(wrapper.findComponent({ name: 'RouterView' }).exists()).toBe(true)
    })
})

