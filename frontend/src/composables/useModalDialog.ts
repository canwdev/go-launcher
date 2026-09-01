import { ref } from 'vue'

export function useModalDialog(onSubmit: (name: string) => void) {
  const open = ref(false)
  const title = ref('')
  const name = ref('')

  function openRename(dialogTitle: string, initialName: string) {
    title.value = dialogTitle
    name.value = initialName
    open.value = true
  }

  function ok() {
    const value = name.value.trim()
    if (value)
      onSubmit(value)
    open.value = false
  }

  function close() {
    open.value = false
  }

  return { open, title, name, openRename, ok, close }
}
