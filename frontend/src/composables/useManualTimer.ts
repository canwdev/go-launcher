import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useStorage } from '@vueuse/core'

/**
 * 前端手动计时器（状态完全由前端维护，useStorage 响应式持久化到 localStorage）：
 * - start(guid)：开始为某个 item 计时（同一时间仅一个计时；若已有则在开始新计时前先回调写入旧的）
 * - stop(onSave)：停止计时，把累计毫秒数交给 onSave 回调（由调用方写入后端）
 * - elapsedMs：当前计时累计毫秒数（computed，每 30s 刷新一次，驱动 runtimeText 的 (+Xm) 显示）
 */

const STORAGE_KEY = 'launcher-manual-timer'
// const AUTO_TIMER_KEY = 'launcher-auto-timer' // autoTimer 特性暂时注释
const TICK_MS = 30_000

interface PersistedTimer {
  guid: string
  startAt: number
}

export function useManualTimer() {
  const activeGuid = ref<string | null>(null)
  const startAt = ref(0)
  // 30s tick：仅用于驱动 elapsedMs 的响应式重算，显示分钟粒度足够。
  const tick = ref(0)
  // 响应式持久化：读写自动同步 localStorage，重启应用后计时继续。
  // 默认值用对象（而非 null）→ useStorage 走 object serializer（JSON 读写），
  // 避免 null 默认值落到 'any' serializer 把对象 String 成 "[object Object]"。
  const persisted = useStorage<PersistedTimer | null>(STORAGE_KEY, { guid: '', startAt: 0 })
  // autoTimer 特性暂时注释：每 item 的"启动后自动触发手动计时"开关（guid → boolean）
  // const autoTimerEnabled = useStorage<Record<string, boolean>>(AUTO_TIMER_KEY, {})

  let interval: number | undefined

  function startInterval() {
    stopInterval()
    interval = window.setInterval(() => {
      tick.value = Date.now()
    }, TICK_MS)
  }

  function stopInterval() {
    if (interval) {
      window.clearInterval(interval)
      interval = undefined
    }
  }

  const elapsedMs = computed(() => {
    // 依赖 tick，使每 30s 刷新一次显示
    void tick.value
    return activeGuid.value ? Math.max(0, Date.now() - startAt.value) : 0
  })

  function isActive(guid: string): boolean {
    return activeGuid.value === guid
  }

  // autoTimer 特性暂时注释：
  // function isAutoTimer(guid: string): boolean {
  //   return autoTimerEnabled.value[guid] ?? false
  // }
  // function setAutoTimer(guid: string, enabled: boolean) {
  //   autoTimerEnabled.value = { ...autoTimerEnabled.value, [guid]: enabled }
  // }

  /**
   * 开始计时。若已有其它 item 在计时：
   * - steal=true（手动触发）：先停止旧计时并把累计时间交给 onStopped 回调；
   * - steal=false（auto 触发）：直接跳过，不抢占正在进行的计时。
   */
  function start(guid: string, onStopped?: (guid: string, ms: number) => void, steal = true) {
    if (activeGuid.value && activeGuid.value !== guid) {
      if (!steal)
        return
      const prevGuid = activeGuid.value
      const prevMs = Math.max(0, Date.now() - startAt.value)
      activeGuid.value = null
      startAt.value = 0
      persisted.value = null
      onStopped?.(prevGuid, prevMs)
    }
    activeGuid.value = guid
    startAt.value = Date.now()
    persisted.value = { guid, startAt: startAt.value }
    startInterval()
  }

  /**
   * 停止计时，把累计毫秒数交给 onSave 回调（调用方负责写入后端 + toast）。
   */
  function stop(onSave: (guid: string, ms: number) => void) {
    if (!activeGuid.value)
      return
    const guid = activeGuid.value
    const ms = Math.max(0, Date.now() - startAt.value)
    activeGuid.value = null
    startAt.value = 0
    persisted.value = null
    stopInterval()
    onSave(guid, ms)
  }

  onMounted(() => {
    const t = persisted.value
    // t.guid truthy：排除从未计时的默认值 { guid: '', startAt: 0 } 与坏数据
    if (t && t.guid && typeof t.startAt === 'number') {
      activeGuid.value = t.guid
      startAt.value = t.startAt
      startInterval()
    }
  })

  onUnmounted(stopInterval)

  // autoTimer 特性暂时注释：不返回 isAutoTimer / setAutoTimer
  return { activeGuid, isActive, elapsedMs, start, stop }
}
