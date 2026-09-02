import { Clock, Timer } from '@lucide/vue'
import { useTimeoutFn } from '@vueuse/core'
import { computed, ref } from 'vue'
import { Launch, Open, Stop } from '../api'
import { formatRuntime, showError } from '../utils'

export interface ItemLaunchState {
  item: { guid: string }
  running: boolean
  /** 展示用 runtime ms（已含 live 计算 / autoTimer baseline 处理） */
  runtimeMs: number
  gameMode: boolean
  timerActive: boolean
  timerMinutes: string
  autoTimer: boolean
}

export interface ItemLaunchCallbacks {
  onLaunched: () => void
  onStopTimer: () => void
  onEditRuntime: () => void
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

  const runtimeText = computed(() => formatRuntime(o.value.runtimeMs))
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
        await Stop(v.item.guid)
      }
      else {
        // 计时中禁止启动新运行
        if (v.timerActive)
          return
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
