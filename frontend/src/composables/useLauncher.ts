import type { LauncherItem } from '../api'
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOff, EventsOn, OnFileDrop } from '../../wailsjs/runtime/runtime'
import { AddPaths, GetItems } from '../api'

export function useLauncher() {
  const items = ref<LauncherItem[]>([])

  async function refresh() {
    try {
      items.value = await GetItems()
    }
    catch (err) {
      console.error(err)
    }
  }

  function onDrop(_x: number, _y: number, paths: string[]) {
    AddPaths(paths).catch((err) => {
      console.error(err)
    })
  }

  onMounted(() => {
    refresh()
    EventsOn('items:updated', refresh)
    OnFileDrop(onDrop, false)
  })

  onUnmounted(() => {
    EventsOff('items:updated')
  })

  return { items, refresh }
}
