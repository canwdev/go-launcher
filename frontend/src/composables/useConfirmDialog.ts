import { ref } from 'vue'

export function useConfirmDialog() {
  const open = ref(false)
  const message = ref('')
  let action: null | (() => void) = null
  let cancelAction: null | (() => void) = null

  function request(msg: string, fn: () => void) {
    message.value = msg
    action = fn
    cancelAction = null
    open.value = true
  }

  // 异步确认：resolve(true)=确认，resolve(false)=取消/关闭
  function requestAsync(msg: string): Promise<boolean> {
    message.value = msg
    open.value = true
    return new Promise((resolve) => {
      action = () => resolve(true)
      cancelAction = () => resolve(false)
    })
  }

  function confirm() {
    open.value = false
    action?.()
    action = null
    cancelAction = null
  }

  function close() {
    open.value = false
    cancelAction?.()
    action = null
    cancelAction = null
  }

  return { open, message, request, requestAsync, confirm, close }
}
