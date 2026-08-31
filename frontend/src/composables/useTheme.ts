import { onMounted, onUnmounted, ref, watch } from 'vue'

export type Theme = 'auto' | 'light' | 'dark'

const STORAGE_KEY = 'launcher-theme'

const validThemes: Theme[] = ['auto', 'light', 'dark']

function loadTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY)
  return (validThemes as string[]).includes(stored ?? '') ? (stored as Theme) : 'auto'
}

function isDark(theme: Theme): boolean {
  if (theme === 'dark')
    return true
  if (theme === 'light')
    return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export function applyStoredTheme() {
  document.documentElement.classList.toggle('dark', isDark(loadTheme()))
}

export function useTheme() {
  const theme = ref<Theme>(loadTheme())
  const dark = ref(false)

  function update() {
    dark.value = isDark(theme.value)
    document.documentElement.classList.toggle('dark', dark.value)
  }

  function setTheme(t: Theme) {
    theme.value = t
    localStorage.setItem(STORAGE_KEY, t)
    update()
  }

  watch(theme, update)

  let media: MediaQueryList | undefined
  onMounted(() => {
    update()
    media = window.matchMedia('(prefers-color-scheme: dark)')
    media.addEventListener('change', update)
  })

  onUnmounted(() => {
    media?.removeEventListener('change', update)
  })

  return { theme, dark, setTheme }
}
