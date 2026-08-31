import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'
import { GetAutoMinimize, SetAutoMinimize } from '../api'

export function useAutoMinimize() {
  const autoMinimize = ref(false)

  async function refresh() {
    try {
      autoMinimize.value = await GetAutoMinimize()
    }
    catch (err) {
      console.error(err)
    }
  }

  async function toggle() {
    try {
      await SetAutoMinimize(!autoMinimize.value)
      autoMinimize.value = !autoMinimize.value
    }
    catch (err) {
      console.error(err)
    }
  }

  onMounted(() => {
    refresh()
    EventsOn('settings:updated', refresh)
  })

  onUnmounted(() => {
    EventsOff('settings:updated')
  })

  return { autoMinimize, toggle, refresh }
}
