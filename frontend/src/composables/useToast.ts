import { reactive } from 'vue'

export interface Toast {
  id: number
  message: string
  type: 'success' | 'error'
}

const toasts = reactive<Toast[]>([])

let nextId = 1

export function showToast(message: string, type: Toast['type'] = 'success') {
  const id = nextId++
  toasts.push({ id, message, type })
  window.setTimeout(() => {
    const idx = toasts.findIndex(t => t.id === id)
    if (idx >= 0)
      toasts.splice(idx, 1)
  }, 3000)
}

export function useToast() {
  return { toasts, showToast }
}
