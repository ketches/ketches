import { toast as sonnerToast } from "sonner"

export interface ToastProps {
  title?: string
  description?: string
  variant?: "default" | "destructive" | "success" | "info" | "warning"
}

export function useToast() {
  return {
    toast: ({ title, description, variant = "default" }: ToastProps) => {
      const options = {
        description: description,
      }

      switch (variant) {
        case "destructive":
          sonnerToast.error(title, options)
          break
        case "success":
          sonnerToast.success(title, options)
          break
        case "warning":
          sonnerToast.warning(title, options)
          break
        case "info":
          sonnerToast.info(title, options)
          break
        default:
          sonnerToast(title, options)
          break
      }
    },
  }
}

export const toast = (props: ToastProps) => {
    const { title, description, variant = "default" } = props
    const options = { description }
    
    switch (variant) {
        case "destructive":
          sonnerToast.error(title, options)
          break
        case "success":
          sonnerToast.success(title, options)
          break
        case "warning":
          sonnerToast.warning(title, options)
          break
        case "info":
          sonnerToast.info(title, options)
          break
        default:
          sonnerToast(title, options)
          break
      }
}
