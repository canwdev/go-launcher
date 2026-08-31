import type { AppItem, AppStore } from '../api'
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOff, EventsOn, OnFileDrop } from '../../wailsjs/runtime/runtime'
import { AddFiles, AddPaths, GetData, SaveData } from '../api'
import { debounce, randomUUID, showError } from '../utils'

export type GridSlot = AppItem | null

export interface StoreTab {
  guid: string
  name: string
  slots: GridSlot[]
}

interface StoreTabSettings {
  auto_minimize: boolean
}

interface Store {
  version: string
  active_tab_guid: string
  tabs: StoreTab[]
  settings: StoreTabSettings
}

function newStore(): Store {
  return {
    version: '1',
    active_tab_guid: '',
    tabs: [],
    settings: { auto_minimize: true },
  }
}

export function useStore() {
  const store = ref<Store>(newStore())

  async function saveNow() {
    try {
      const payload = JSON.parse(JSON.stringify(store.value)) as AppStore
      for (const tab of payload.tabs) {
        for (const slot of tab.slots ?? []) {
          if (!slot)
            continue
          const rec = slot as unknown as Record<string, unknown>
          delete rec.iconURL
          delete rec.running
        }
      }
      await SaveData(payload)
    }
    catch (err) {
      showError(err)
    }
  }

  const save = debounce(saveNow as () => void, 300)

  async function refresh() {
    try {
      const data = await GetData()
      store.value = data as unknown as Store
      for (const tab of store.value.tabs) {
        if (!tab.slots)
          tab.slots = []
      }
      forceActiveTab()
    }
    catch (err) {
      console.error(err)
    }
  }

  const activeTab = ref<StoreTab | null>(null)

  function forceActiveTab() {
    let tab = store.value.tabs.find(t => t.guid === store.value.active_tab_guid) ?? null
    if (!tab && store.value.tabs.length)
      tab = store.value.tabs[0]!
    activeTab.value = tab
    if (tab && !store.value.active_tab_guid)
      store.value.active_tab_guid = tab.guid
  }

  async function setActiveTab(guid: string) {
    store.value.active_tab_guid = guid
    activeTab.value = store.value.tabs.find(t => t.guid === guid) ?? null
    await save()
  }

  function ensureActive() {
    if (!activeTab.value && store.value.tabs.length) {
      const t = store.value.tabs[0]!
      activeTab.value = t
      if (!store.value.active_tab_guid)
        store.value.active_tab_guid = t.guid
    }
  }

  async function addTab(name = 'New Tab') {
    const tab: StoreTab = { guid: randomUUID(), name, slots: [] }
    store.value.tabs.push(tab)
    await setActiveTab(tab.guid)
  }

  async function removeTab(guid: string) {
    const idx = store.value.tabs.findIndex(t => t.guid === guid)
    if (idx < 0)
      return
    store.value.tabs.splice(idx, 1)
    if (store.value.active_tab_guid === guid) {
      const next = store.value.tabs[Math.min(idx, store.value.tabs.length - 1)] ?? null
      store.value.active_tab_guid = next?.guid ?? ''
      activeTab.value = next
    }
    await save()
  }

  async function renameTab(guid: string, name: string) {
    const tab = store.value.tabs.find(t => t.guid === guid)
    if (!tab)
      return
    tab.name = name
    await save()
  }

  async function moveTab(from: number, to: number) {
    if (from < 0 || from >= store.value.tabs.length || to < 0 || to >= store.value.tabs.length)
      return
    if (from === to)
      return
    const [t] = store.value.tabs.splice(from, 1)
    store.value.tabs.splice(to, 0, t!)
    if (store.value.active_tab_guid === t!.guid)
      activeTab.value = t!
    await save()
  }

  async function addItems(items: AppItem[]) {
    if (!items.length)
      return
    ensureActive()
    if (!activeTab.value) {
      await addTab('Default')
      ensureActive()
    }
    if (!activeTab.value)
      return
    for (const item of items) {
      activeTab.value.slots.push(item)
    }
    await save()
  }

  async function addFilesInto() {
    try {
      const items = await AddFiles()
      if (items.length)
        addItems(items)
    }
    catch (err) {
      showError(err)
    }
  }

  async function removeItem(guid: string) {
    ensureActive()
    if (!activeTab.value)
      return
    const idx = activeTab.value.slots.findIndex(s => s?.guid === guid)
    if (idx >= 0) {
      activeTab.value.slots[idx] = null
      await save()
    }
  }

  async function renameItem(guid: string, name: string) {
    for (const tab of store.value.tabs) {
      const slot = tab.slots.find(s => s?.guid === guid)
      if (slot) {
        slot.name = name
        await save()
        return
      }
    }
  }

  async function updateItemIcon(guid: string, icon: string) {
    for (const tab of store.value.tabs) {
      const slot = tab.slots.find(s => s?.guid === guid)
      if (slot) {
        slot.icon = icon
        slot.iconURL = ''
        await save()
        await refresh()
        return
      }
    }
  }

  async function setAutoMinimize(enabled: boolean) {
    store.value.settings.auto_minimize = enabled
    await save()
  }

  function onDrop(_x: number, _y: number, paths: string[]) {
    AddPaths(paths)
      .then((items) => {
        if (items.length)
          addItems(items)
      })
      .catch(showError)
  }

  onMounted(() => {
    refresh()
    EventsOn('items:updated', refresh)
    OnFileDrop(onDrop, false)
  })

  onUnmounted(() => {
    EventsOff('items:updated')
  })

  return {
    store,
    activeTab,
    refresh,
    save,
    setActiveTab,
    addTab,
    removeTab,
    renameTab,
    moveTab,
    addItems,
    addFilesInto,
    removeItem,
    renameItem,
    updateItemIcon,
    setAutoMinimize,
  }
}
