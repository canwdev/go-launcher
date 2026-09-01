import { ref } from 'vue'

export function useConfirmDialog() {
  const open = ref(false)
  const message = ref('')
  let action: null | (() => void) = null

  function request(msg: string, fn: () => void) {
    message.value = msg
    action = fn
    open.value = true
  }

  function confirm() {
    open.value = false
    action?.()
    action = null
  }

  function close() {
    open.value = false
    action = null
  }

  return { open, message, request, confirm, close }
}
