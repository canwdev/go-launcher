import type { useManualTimer } from './useManualTimer'
import { onMounted, onUnmounted, ref } from 'vue'
import { Launch, Open } from '../api'
import { showError } from '../utils'

/**
 * 全局搜索浮层：打开状态 + 启动回调 + Ctrl/Cmd+F 快捷键。
 * 从 App.vue 抽离，保持主组件精简。
 */
export function useSearch(ctx: {
  setActiveTab: (guid: string) => void
  timer: ReturnType<typeof useManualTimer>
}) {
  const searchOpen = ref(false)

  const searchOnLaunch = (guid: string, tabGuid: string) => {
    ctx.setActiveTab(tabGuid)
    const p = ctx.timer.isAutoTimer(guid) ? Open(guid) : Launch(guid)
    p.catch(showError)
  }

  // Ctrl+F 打开搜索（仅应用内生效，非系统全局热键）
  function onGlobalKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
      e.preventDefault()
      if (!searchOpen.value)
        searchOpen.value = true
    }
  }

  onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))

  return { searchOpen, searchOnLaunch }
}
