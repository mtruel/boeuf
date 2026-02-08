import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useToast } from '../useToast'

// Mock vue-sonner
vi.mock('vue-sonner', () => ({
    toast: {
        success: vi.fn(),
        error: vi.fn(),
        warning: vi.fn(),
        info: vi.fn(),
        dismiss: vi.fn(),
    }
}))

describe('useToast', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should call toast.success with message', async () => {
        const { success } = useToast()
        const { toast } = await import('vue-sonner')

        success('Test success message')

        expect(toast.success).toHaveBeenCalledWith('Test success message', { description: undefined })
    })

    it('should call toast.error with message and description', async () => {
        const { error } = useToast()
        const { toast } = await import('vue-sonner')

        error('Test error', 'Error description')

        expect(toast.error).toHaveBeenCalledWith('Test error', { description: 'Error description' })
    })

    it('should call toast.warning with message', async () => {
        const { warning } = useToast()
        const { toast } = await import('vue-sonner')

        warning('Test warning')

        expect(toast.warning).toHaveBeenCalledWith('Test warning', { description: undefined })
    })

    it('should dismiss toasts', async () => {
        const { dismiss } = useToast()
        const { toast } = await import('vue-sonner')

        dismiss('test-id')

        expect(toast.dismiss).toHaveBeenCalledWith('test-id')
    })
})
