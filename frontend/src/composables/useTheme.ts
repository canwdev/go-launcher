import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useStorage } from '@vueuse/core'

export type Theme = 'auto' | 'light' | 'dark'

const STORAGE_KEY = 'launcher-theme'

const validThemes: Theme[] = ['auto', 'light', 'dark']

// 模块级响应式持久化：main.ts 挂载前也能读取（避免主题闪烁），写入自动同步 localStorage
const storedTheme = useStorage<Theme>(STORAGE_KEY, 'auto')

// 若历史存储值非法，归一到默认值
if (!validThemes.includes(storedTheme.value))
  storedTheme.value = 'auto'

function isDark(theme: Theme): boolean {
  if (theme === 'dark')
    return true
  if (theme === 'light')
    return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export function applyStoredTheme() {
  document.documentElement.classList.toggle('dark', isDark(storedTheme.value))
}

export function useTheme() {
  const theme = storedTheme
  const dark = ref(false)

  function update() {
    dark.value = isDark(theme.value)
    document.documentElement.classList.toggle('dark', dark.value)
  }

  function setTheme(t: Theme) {
    theme.value = t // useStorage 自动写回 localStorage
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
