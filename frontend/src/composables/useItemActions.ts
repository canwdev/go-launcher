import { Clock, Timer } from '@lucide/vue'
import { useTimeoutFn } from '@vueuse/core'
import { computed, ref } from 'vue'
import { Launch, Open, Stop } from '../api'
import { formatRuntime, showError } from '../utils'

export interface ItemLaunchState {
  item: { guid: string }
  running: boolean
  /** 累计展示用时 baseline ms（不含本次会话增量；autoTimer 项同样为 baseline） */
  baselineMs: number
  /** 本次会话计时增量 ms（手动计时 elapsed 或自动计时 live 增量）；0=未计时 */
  liveMs: number
  gameMode: boolean
  timerActive: boolean
  autoTimer: boolean
}

export interface ItemLaunchCallbacks {
  onLaunched: () => void
  onStopTimer: () => void
  onEditRuntime: () => void
  /** 可选：停止前确认（grid 视图点击运行中程序需弹窗确认）。返回 false 则取消停止。 */
  confirmStop?: () => Promise<boolean>
}

/**
 * 抽取 item 启动 / runtime 展示的通用逻辑，供 LauncherRow（列表）与 GridItem（网格）复用：
 * - onRun：running→Stop；否则 autoTimer ? Open : Launch（不跟踪进程），成功后 onLaunched
 * - onClick：runtime 点击——计时中→停止；运行中→忽略；否则→编辑 runtime
 * - runtimeText / runtimeIcon：展示串与图标（autoTimer 用 Timer，否则 Clock）
 */
const LAUNCH_COOLDOWN_MS = 1000

export function useItemActions(
  get: () => ItemLaunchState,
  callbacks: ItemLaunchCallbacks,
) {
  const o = computed(get)

  // 启动 1s 冷却：防止双击 / 重复点击导致重复启动
  const launchCooling = ref(false)
  const { start: startLaunchCooldown } = useTimeoutFn(() => {
    launchCooling.value = false
  }, LAUNCH_COOLDOWN_MS, { immediate: false })

  const runtimeText = computed(() => {
    const base = formatRuntime(o.value.baselineMs)
    // Manual / Auto 计时统一为 "Xm (+ Xm)" 字符串（无图标）：
    if (o.value.liveMs > 0)
      return `${base} (+ ${formatRuntime(o.value.liveMs)})`
    return base
  })
  const runtimeIcon = computed(() => (o.value.autoTimer ? Timer : Clock))

  async function onRun() {
    if (launchCooling.value)
      return
    launchCooling.value = true
    startLaunchCooldown()
    const v = o.value
    try {
      if (v.running) {
        // 计时中允许 Stop（否则 autoTimer 启动后程序将无法停止）
        // grid 视图可要求停止前确认：confirmStop 返回 false 则取消
        if (callbacks.confirmStop && !(await callbacks.confirmStop()))
          return
        await Stop(v.item.guid)
      }
      else {
        // 手动计时进行中也可再次启动（不再禁用）；autoTimer 启动会自动触发计时
        if (v.autoTimer)
          await Open(v.item.guid)
        else
          await Launch(v.item.guid)
        callbacks.onLaunched()
      }
    }
    catch (err) {
      showError(err)
    }
  }

  function onClick() {
    const v = o.value
    if (v.timerActive) {
      callbacks.onStopTimer()
      return
    }
    if (v.running)
      return
    callbacks.onEditRuntime()
  }

  return { runtimeText, runtimeIcon, onRun, onClick }
}
