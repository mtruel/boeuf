import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import App from './App.vue'

describe('App', () => {
    it('should mount and render the app title', () => {
        const wrapper = mount(App)
        expect(wrapper.find('h1').text()).toContain('Boeuf')
    })

    it('should display backend health status section', () => {
        const wrapper = mount(App)
        expect(wrapper.text()).toContain('État du Backend')
    })

    it('should render shadcn-vue Button component', () => {
        const wrapper = mount(App)
        const button = wrapper.find('button')
        expect(button.exists()).toBe(true)
        expect(button.text()).toContain('Actualiser')
    })

    it('should call health check on mount', async () => {
        const fetchSpy = vi.spyOn(globalThis, 'fetch')
        mount(App)

        // Wait for onMounted to execute
        await new Promise(resolve => setTimeout(resolve, 0))

        expect(fetchSpy).toHaveBeenCalledWith('/api/health')
    })
})
