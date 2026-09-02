import { onMounted, onUnmounted, ref } from 'vue'

/**
 * 前端统一"自动计时"（进程运行时长）：
 * 后端只推送 baseline(runtime_ms) + 进程启动时间(start_at)；前端负责
 * 运行中增量计算「baseline + (now - start_at)」并每 30s 刷新显示。
 * 进程退出后由后端兜底累计（exit goroutine），前端显示自然回落回 baseline。
 *
 * 与 useManualTimer（手动计时，用户主观记录）相互独立，可同时存在。
 */

const TICK_MS = 30_000

export function useAutoRuntime() {
  const now = ref(0)
  let interval: number | undefined

  onMounted(() => {
    interval = window.setInterval(() => {
      now.value = Date.now()
    }, TICK_MS)
  })

  onUnmounted(() => {
    if (interval) {
      window.clearInterval(interval)
      interval = undefined
    }
  })

  /**
   * 运行中显示毫秒数 = baseline + 已运行增量；未运行则返回 baseline。
   * 读取 now 触发依赖收集，使每 30s tick 重渲染刷新显示。
   */
  function liveMs(baseline: number, running: boolean, startAt: number): number {
    void now.value
    if (!running || startAt <= 0)
      return baseline
    const live = baseline + (Date.now() - startAt)
    return live > baseline ? live : baseline
  }

  return { liveMs }
}
