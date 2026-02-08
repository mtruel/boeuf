import { toast } from 'vue-sonner'

export interface ToastAction {
    label: string
    onClick: () => void
}

export function useToast() {
    return {
        success: (message: string, description?: string) => {
            toast.success(message, { description })
        },

        error: (message: string, description?: string) => {
            toast.error(message, { description })
        },

        warning: (message: string, description?: string) => {
            toast.warning(message, { description })
        },

        info: (message: string, description?: string) => {
            toast.info(message, { description })
        },

        // Custom toast with actions
        custom: (
            message: string,
            description?: string,
            options?: {
                duration?: number
                action?: ToastAction
                cancel?: ToastAction
            }
        ) => {
            toast(message, {
                description,
                duration: options?.duration,
                action: options?.action,
                cancel: options?.cancel
            })
        },

        // Dismiss a specific toast or all
        dismiss: (toastId?: string | number) => {
            toast.dismiss(toastId)
        }
    }
}
