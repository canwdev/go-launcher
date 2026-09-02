import type { AppData, AppItem, AppStore, ItemState } from '../api'
import { onMounted, onUnmounted, ref } from 'vue'
import { useStorage } from '@vueuse/core'
import { EventsOff, EventsOn, OnFileDrop } from '../../wailsjs/runtime/runtime'
import { AddFiles, AddPaths, ConvertToAbsolute, ConvertToRelative, GetData, SaveData, SetRuntimeMs, UpdateIcon } from '../api'
import { debounce, isAutoIcon, randomUUID, showError } from '../utils'
import { showToast } from './useToast'

export type GridSlot = string | null

export interface Category {
  guid: string
  name: string
  slots: GridSlot[]
}

export interface StoreSettings {
  game_mode: boolean
  absolute_paths: boolean
}

export interface Store {
  apps: Record<string, AppItem>
  categories: Category[]
  settings: StoreSettings
}

const ACTIVE_TAB_KEY = 'launcher-active-tab'

function newStore(): Store {
  return {
    apps: {},
    categories: [],
    settings: { game_mode: true, absolute_paths: true },
  }
}

export function useStore() {
  const store = ref<Store>(newStore())
  const state = ref<Record<string, ItemState>>({})
  const activeTab = ref<Category | null>(null)
  // 响应式持久化：当前激活 tab 自动同步 localStorage
  const savedActiveTab = useStorage<string>(ACTIVE_TAB_KEY, '')

  function applyData(data: AppData) {
    store.value = data.store as unknown as Store
    state.value = data.state
    for (const cat of store.value.categories) {
      if (!cat.slots)
        cat.slots = []
    }
    forceActiveTab()
  }

  async function refresh() {
    try {
      applyData(await GetData())
    }
    catch (err) {
      console.error(err)
    }
  }

  function saveNow() {
    return SaveData(store.value as unknown as AppStore).catch(showError)
  }

  const save = debounce(saveNow, 300)

  function forceActiveTab() {
    const saved = savedActiveTab.value
    let tab = store.value.categories.find(c => c.guid === saved) ?? null
    if (!tab && store.value.categories.length)
      tab = store.value.categories[0]!
    activeTab.value = tab
    if (tab)
      savedActiveTab.value = tab.guid
  }

  function ensureActive() {
    if (!activeTab.value && store.value.categories.length) {
      const t = store.value.categories[0]!
      activeTab.value = t
      savedActiveTab.value = t.guid
    }
  }

  function setActiveTab(guid: string) {
    activeTab.value = store.value.categories.find(c => c.guid === guid) ?? null
    if (activeTab.value)
      savedActiveTab.value = guid
  }

  async function addTab(name = 'New Tab') {
    const cat: Category = { guid: randomUUID(), name, slots: [] }
    store.value.categories.push(cat)
    setActiveTab(cat.guid)
  }

  async function removeTab(guid: string) {
    const idx = store.value.categories.findIndex(c => c.guid === guid)
    if (idx < 0)
      return
    store.value.categories.splice(idx, 1)
    if (activeTab.value?.guid === guid) {
      const next = store.value.categories[Math.min(idx, store.value.categories.length - 1)] ?? null
      activeTab.value = next
      savedActiveTab.value = next?.guid ?? ''
    }
    pruneOrphans()
    await save()
  }

  async function renameTab(guid: string, name: string) {
    const cat = store.value.categories.find(c => c.guid === guid)
    if (!cat)
      return
    cat.name = name
    await save()
  }

  async function moveTab(from: number, to: number) {
    if (from < 0 || from >= store.value.categories.length || to < 0 || to >= store.value.categories.length)
      return
    if (from === to)
      return
    const [t] = store.value.categories.splice(from, 1)
    store.value.categories.splice(to, 0, t!)
    if (activeTab.value?.guid === t!.guid)
      activeTab.value = t!
    await save()
  }

  async function addItems(items: AppItem[], icons: Record<string, string>) {
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
      store.value.apps[item.guid] = item
      if (icons[item.guid])
        state.value[item.guid] = { ...state.value[item.guid], icon_url: icons[item.guid] }
      activeTab.value.slots.push(item.guid)
    }
    await save()
  }

  async function addFilesInto() {
    try {
      const res = await AddFiles()
      if (res.items.length)
        await addItems(res.items, res.icons)
    }
    catch (err) {
      showError(err)
    }
  }

  function pruneOrphans() {
    for (const cat of store.value.categories)
      cat.slots = cat.slots.filter((s): s is string => s != null)
    const referenced = new Set(store.value.categories.flatMap(c => c.slots))
    for (const guid of Object.keys(store.value.apps)) {
      if (!referenced.has(guid)) {
        delete store.value.apps[guid]
        delete state.value[guid]
      }
    }
  }

  async function removeItem(guid: string) {
    for (const cat of store.value.categories)
      cat.slots = cat.slots.filter(s => s !== guid)
    pruneOrphans()
    await save()
  }

  async function renameItem(guid: string, name: string) {
    const app = store.value.apps[guid]
    if (!app)
      return
    app.name = name
    await save()
  }

  async function duplicateItem(guid: string) {
    const app = store.value.apps[guid]
    const tab = activeTab.value
    if (!app || !tab)
      return
    const copy: AppItem = {
      ...app,
      guid: randomUUID(),
      name: `${app.name} (copy)`,
      runtime_ms: 0,
    }
    store.value.apps[copy.guid] = copy
    if (state.value[guid]?.icon_url)
      state.value[copy.guid] = { ...state.value[copy.guid], icon_url: state.value[guid].icon_url }
    const idx = tab.slots.indexOf(guid)
    if (idx >= 0)
      tab.slots.splice(idx + 1, 0, copy.guid)
    else
      tab.slots.push(copy.guid)
    await save()
  }

  async function moveItemToTab(guid: string, toTabGuid: string) {
    const target = store.value.categories.find(c => c.guid === toTabGuid)
    if (!target)
      return
    for (const cat of store.value.categories)
      cat.slots = cat.slots.filter(s => s !== guid)
    target.slots.push(guid)
    await save()
  }

  async function copyItemToTab(guid: string, toTabGuid: string) {
    const app = store.value.apps[guid]
    const target = store.value.categories.find(c => c.guid === toTabGuid)
    if (!app || !target)
      return
    const copy: AppItem = {
      ...app,
      guid: randomUUID(),
      name: `${app.name} (copy)`,
      runtime_ms: 0,
    }
    store.value.apps[copy.guid] = copy
    if (state.value[guid]?.icon_url)
      state.value[copy.guid] = { ...state.value[copy.guid], icon_url: state.value[guid].icon_url }
    target.slots.push(copy.guid)
    await save()
  }

  async function updateItemIcon(guid: string, icon: string, iconUrl: string, silent = false) {
    const app = store.value.apps[guid]
    if (!app)
      return
    app.icon = icon
    state.value[guid] = { ...state.value[guid], icon_url: iconUrl }
    await save()
    if (!silent)
      showToast('Icon updated')
  }

  async function updateItem(guid: string, fields: Partial<AppItem>) {
    const app = store.value.apps[guid]
    if (!app)
      return
    Object.assign(app, fields)
    await saveNow()
    await refresh()
    showToast('Item updated')
  }

  async function batchUpdateIcons() {
    let count = 0
    for (const app of Object.values(store.value.apps)) {
      if (!isAutoIcon(app.icon))
        continue
      try {
        const res = await UpdateIcon(app.guid)
        await updateItemIcon(app.guid, res.icon, res.icon_url, true)
        count++
      }
      catch {
        // skip items whose icon cannot be regenerated
      }
    }
    showToast(`Updated ${count} icon${count === 1 ? '' : 's'}`)
  }

  async function setGameMode(enabled: boolean) {
    store.value.settings.game_mode = enabled
    await save()
  }

  async function setRuntimeMs(guid: string, ms: number) {
    await SetRuntimeMs(guid, Math.max(0, Math.floor(ms)))
    await refresh()
  }

  async function setAbsolutePaths(enabled: boolean) {
    store.value.settings.absolute_paths = enabled
    await save()
  }

  async function convertToAbsolute() {
    try {
      await ConvertToAbsolute()
      await refresh()
      showToast('Converted to absolute path')
    }
    catch (err) {
      showError(err)
    }
  }

  async function convertToRelative() {
    try {
      await ConvertToRelative()
      await refresh()
      showToast('Converted to relative path')
    }
    catch (err) {
      showError(err)
    }
  }

  function onDrop(_x: number, _y: number, paths: string[]) {
    AddPaths(paths)
      .then((res) => {
        if (res.items.length)
          return addItems(res.items, res.icons)
      })
      .catch(showError)
  }

  onMounted(() => {
    refresh()
    EventsOn('state:updated', (st: Record<string, ItemState>) => {
      state.value = st
    })
    OnFileDrop(onDrop, false)
  })

  onUnmounted(() => {
    EventsOff('state:updated')
  })

  return {
    store,
    state,
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
    duplicateItem,
    moveItemToTab,
    copyItemToTab,
    updateItemIcon,
    updateItem,
    batchUpdateIcons,
    setGameMode,
    setRuntimeMs,
    setAbsolutePaths,
    convertToAbsolute,
    convertToRelative,
  }
}
