import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import JoinSessionView from './JoinSessionView.vue'
import { nextTick } from 'vue'

// Mock fetch globally
global.fetch = vi.fn()

describe('JoinSessionView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should display loading state initially', () => {
    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'test-token-123'
      }
    })

    expect(wrapper.text()).toContain('Rejoindre la session')
  })

  it('should call join API with invite token', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ sessionId: 'sess_123' })
    })
    global.fetch = mockFetch

    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'test-token-123'
      }
    })

    await flushPromises()

    expect(mockFetch).toHaveBeenCalledWith('/api/sessions/join', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ inviteToken: 'test-token-123' })
    })
  })

  it('should display success message after successful join', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ sessionId: 'sess_abc123' })
    })
    global.fetch = mockFetch

    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'valid-token'
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Session rejointe')
    expect(wrapper.text()).toContain('sess_abc123')
  })

  it('should display error when token is invalid', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({ 
        code: 'SESSION_NOT_FOUND', 
        message: 'Invalid or expired invitation' 
      })
    })
    global.fetch = mockFetch

    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'invalid-token'
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Erreur')
    expect(wrapper.text()).toContain('Invalid or expired invitation')
  })

  it('should redirect to login if unauthenticated', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ 
        code: 'UNAUTHENTICATED', 
        message: 'Authentication required' 
      })
    })
    global.fetch = mockFetch

    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'test-token'
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Connecter Spotify')
  })

  it('should handle network errors gracefully', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('Network error'))
    global.fetch = mockFetch

    const wrapper = mount(JoinSessionView, {
      props: {
        token: 'test-token'
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Erreur')
    expect(wrapper.text()).toContain('Network error')
  })
})
