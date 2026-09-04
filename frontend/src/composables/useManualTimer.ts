import { useStorage } from '@vueuse/core'
import { onMounted, ref } from 'vue'
import { useClockTick } from './useClockTick'

/**
 * 前端手动计时器（状态完全由前端维护，useStorage 响应式持久化到 localStorage）：
 * - start(guid)：开始为某个 item 计时；多 item 可并行计时，互不抢占
 * - stop(guid, onSave)：停止指定 item 的计时，把累计毫秒数交给 onSave 回调（由调用方写入后端）
 * - elapsedMs(guid)：指定 item 的累计毫秒数（每 10s 刷新一次，驱动 runtimeText 的 (+Xm) 显示）
 * - isAutoTimer / setAutoTimer：每 item 的"启动后自动触发手动计时"开关（guid → boolean）
 *
 * tick 由 useClockTick 全局共享（与 useAutoRuntime 共用），不再各自维护 interval。
 */

// 多计时并行格式（{ guid: startAt }）
const TIMERS_KEY = 'launcher-manual-timers'
const AUTO_TIMER_KEY = 'launcher-auto-timer'

export function useManualTimer() {
  // guid → startAt(ms)，多 item 可并行计时。
  // 默认值用对象（而非 null）→ useStorage 走 object serializer（JSON 读写），
  // 避免 null 默认值落到 'any' serializer 把对象 String 成 "[object Object]"。
  const timers = ref<Record<string, number>>({})
  // 10s tick：仅用于驱动 elapsedMs 的响应式重算，显示分钟粒度足够。
  const tick = useClockTick()
  const persisted = useStorage<Record<string, number>>(TIMERS_KEY, {})
  const autoTimerEnabled = useStorage<Record<string, boolean>>(AUTO_TIMER_KEY, {})

  function persist() {
    persisted.value = { ...timers.value }
  }

  function isActive(guid: string): boolean {
    return timers.value[guid] != null
  }

  function elapsedMs(guid: string): number {
    // 依赖 tick，使每 10s 刷新一次显示
    void tick.value
    const s = timers.value[guid]
    return s ? Math.max(0, Date.now() - s) : 0
  }

  function isAutoTimer(guid: string): boolean {
    return autoTimerEnabled.value[guid] ?? false
  }

  function setAutoTimer(guid: string, enabled: boolean) {
    autoTimerEnabled.value = { ...autoTimerEnabled.value, [guid]: enabled }
  }

  /**
   * 开始为指定 item 计时。已在计时则不重置（auto 触发时避免重置正在进行的计时）。
   * 多计时并行：不影响其它 item 的计时。
   */
  function start(guid: string) {
    if (timers.value[guid] == null) {
      timers.value = { ...timers.value, [guid]: Date.now() }
      persist()
    }
  }

  /**
   * 停止指定 item 的计时，把累计毫秒数交给 onSave 回调（调用方负责写入后端 + toast）。
   */
  function stop(guid: string, onSave: (ms: number) => void) {
    const s = timers.value[guid]
    if (s == null)
      return
    const ms = Math.max(0, Date.now() - s)
    const next = { ...timers.value }
    delete next[guid]
    timers.value = next
    persist()
    onSave(ms)
  }

  onMounted(() => {
    const saved = persisted.value
    if (saved && typeof saved === 'object') {
      const valid: Record<string, number> = {}
      for (const [g, s] of Object.entries(saved)) {
        if (typeof s === 'number' && s > 0)
          valid[g] = s
      }
      if (Object.keys(valid).length > 0)
        timers.value = valid
    }
  })

  return { isActive, isAutoTimer, setAutoTimer, elapsedMs, start, stop }
}
