import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CreateSessionComponent from './CreateSessionComponent.vue'

// Mock fetch
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('CreateSessionComponent', () => {
    beforeEach(() => {
        mockFetch.mockClear()
    })

    it('should render create session button', () => {
        const wrapper = mount(CreateSessionComponent)

        const button = wrapper.find('[data-testid="create-session-btn"]')
        expect(button.exists()).toBe(true)
        expect(button.text()).toBe('Créer une session')
    })

    it('should show loading state when creating session', async () => {
        // Mock successful response
        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({
                sessionId: 'sess_123',
                inviteUrl: 'http://localhost:3000/join/token123',
                inviteCode: '',
                expiresAt: '2026-01-28T21:00:00Z'
            })
        })

        const wrapper = mount(CreateSessionComponent)
        const button = wrapper.find('[data-testid="create-session-btn"]')

        await button.trigger('click')

        // Check loading state
        expect(button.text()).toContain('Création...')
        expect(button.attributes('disabled')).toBeDefined()
    })

    it('should display invite link and code after successful creation', async () => {
        const mockResponse = {
            sessionId: 'sess_123',
            inviteUrl: 'http://localhost:3000/join/token123',
            inviteCode: '',
            expiresAt: '2026-01-28T21:00:00Z'
        }

        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () => mockResponse
        })

        const wrapper = mount(CreateSessionComponent)
        const button = wrapper.find('[data-testid="create-session-btn"]')

        await button.trigger('click')

        // Wait for the component to update
        await new Promise(resolve => setTimeout(resolve, 0))
        await wrapper.vm.$nextTick()

        // Check that invite URL is displayed
        const inviteLink = wrapper.find('[data-testid="invite-url"]')
        expect(inviteLink.exists()).toBe(true)
        expect(inviteLink.element.value).toBe(mockResponse.inviteUrl)

        // Check copy button exists
        const copyBtn = wrapper.find('[data-testid="copy-btn"]')
        expect(copyBtn.exists()).toBe(true)
    })

    it('should show error message on failed creation', async () => {
        mockFetch.mockResolvedValueOnce({
            ok: false,
            status: 401,
            json: async () => ({
                code: 'UNAUTHENTICATED',
                message: 'Authentication required'
            })
        })

        const wrapper = mount(CreateSessionComponent)
        const button = wrapper.find('[data-testid="create-session-btn"]')

        await button.trigger('click')
        await wrapper.vm.$nextTick()

        // Check error message
        const errorMsg = wrapper.find('[data-testid="error-message"]')
        expect(errorMsg.exists()).toBe(true)
        expect(errorMsg.text()).toContain('Authentication required')
    })

    it('should handle copy to clipboard', async () => {
        // Mock clipboard API
        const mockClipboard = {
            writeText: vi.fn().mockResolvedValue(undefined)
        }
        Object.assign(navigator, { clipboard: mockClipboard })

        const mockResponse = {
            sessionId: 'sess_123',
            inviteUrl: 'http://localhost:3000/join/token123',
            inviteCode: '',
            expiresAt: '2026-01-28T21:00:00Z'
        }

        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () => mockResponse
        })

        const wrapper = mount(CreateSessionComponent)

        // Create session first
        await wrapper.find('[data-testid="create-session-btn"]').trigger('click')
        await wrapper.vm.$nextTick()

        // Click copy button
        await wrapper.find('[data-testid="copy-btn"]').trigger('click')

        expect(mockClipboard.writeText).toHaveBeenCalledWith(mockResponse.inviteUrl)

        // Check success feedback
        const copyFeedback = wrapper.find('[data-testid="copy-feedback"]')
        expect(copyFeedback.exists()).toBe(true)
        expect(copyFeedback.text()).toContain('Copié!')
    })

    it('should handle invalid date format gracefully', async () => {
        const mockResponse = {
            sessionId: 'sess_123',
            inviteUrl: 'http://localhost:3000/join/token123',
            inviteCode: '',
            expiresAt: 'invalid-date-string'
        }

        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () => mockResponse
        })

        const wrapper = mount(CreateSessionComponent)

        // Create session with invalid date
        await wrapper.find('[data-testid="create-session-btn"]').trigger('click')
        await wrapper.vm.$nextTick()

        // Check that component still renders but shows error message for date
        const expirationDate = wrapper.find('[data-testid="expiration-date"]')
        expect(expirationDate.exists()).toBe(true)
        expect(expirationDate.text()).toContain('Date invalide')
    })

    it('should use fallback copy method when clipboard API fails', async () => {
        // Mock clipboard API to throw error
        const mockClipboard = {
            writeText: vi.fn().mockRejectedValue(new Error('Clipboard API not available'))
        }
        Object.assign(navigator, { clipboard: mockClipboard })

        // Mock document.execCommand
        const mockExecCommand = vi.fn().mockReturnValue(true)
        document.execCommand = mockExecCommand

        const mockResponse = {
            sessionId: 'sess_123',
            inviteUrl: 'http://localhost:3000/join/token123',
            inviteCode: '',
            expiresAt: '2026-01-28T21:00:00Z'
        }

        mockFetch.mockResolvedValueOnce({
            ok: true,
            json: async () => mockResponse
        })

        const wrapper = mount(CreateSessionComponent)

        // Create session first
        await wrapper.find('[data-testid="create-session-btn"]').trigger('click')
        await wrapper.vm.$nextTick()

        // Click copy button (should use fallback)
        await wrapper.find('[data-testid="copy-btn"]').trigger('click')
        await wrapper.vm.$nextTick()

        // Verify fallback was used
        expect(mockExecCommand).toHaveBeenCalledWith('copy')

        // Check success feedback still appears
        const copyFeedback = wrapper.find('[data-testid="copy-feedback"]')
        expect(copyFeedback.exists()).toBe(true)
    })
})