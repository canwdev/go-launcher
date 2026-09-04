import { onMounted, onUnmounted, ref } from 'vue'

/**
 * 全局共享的 10s tick：驱动「运行时长」类显示的响应式刷新。
 * useAutoRuntime（自动计时）与 useManualTimer（手动计时）共用同一个 interval，
 * 避免两处各自 setInterval 造成重复开销。
 *
 * 用法：const tick = useClockTick()；在计算属性/函数里读取 tick.value 触发依赖收集。
 */

const TICK_MS = 10_000

const tick = ref(0)

let interval: number | undefined
let subscribers = 0

function subscribe() {
  subscribers++
  if (!interval) {
    interval = window.setInterval(() => {
      tick.value = Date.now()
    }, TICK_MS)
  }
  return () => {
    subscribers = Math.max(0, subscribers - 1)
    if (subscribers === 0 && interval) {
      window.clearInterval(interval)
      interval = undefined
    }
  }
}

export function useClockTick() {
  onMounted(() => {
    const unsubscribe = subscribe()
    onUnmounted(unsubscribe)
  })
  return tick
}
