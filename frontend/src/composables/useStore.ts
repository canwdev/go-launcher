import type { AppData, AppItem, AppStore, ItemState } from '../api'
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOff, EventsOn, OnFileDrop } from '../../wailsjs/runtime/runtime'
import { AddFiles, AddPaths, ConvertToAbsolute, ConvertToRelative, GetData, SaveData } from '../api'
import { debounce, randomUUID, showError } from '../utils'
import { showToast } from './useToast'

export type GridSlot = string | null

export interface Category {
  guid: string
  name: string
  slots: GridSlot[]
}

export interface StoreSettings {
  auto_minimize: boolean
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
    settings: { auto_minimize: true, absolute_paths: true },
  }
}

function loadActiveTab(): string {
  return localStorage.getItem(ACTIVE_TAB_KEY) ?? ''
}

export function useStore() {
  const store = ref<Store>(newStore())
  const state = ref<Record<string, ItemState>>({})
  const activeTab = ref<Category | null>(null)

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
    const saved = loadActiveTab()
    let tab = store.value.categories.find(c => c.guid === saved) ?? null
    if (!tab && store.value.categories.length)
      tab = store.value.categories[0]!
    activeTab.value = tab
    if (tab)
      localStorage.setItem(ACTIVE_TAB_KEY, tab.guid)
  }

  function ensureActive() {
    if (!activeTab.value && store.value.categories.length) {
      const t = store.value.categories[0]!
      activeTab.value = t
      localStorage.setItem(ACTIVE_TAB_KEY, t.guid)
    }
  }

  function setActiveTab(guid: string) {
    activeTab.value = store.value.categories.find(c => c.guid === guid) ?? null
    if (activeTab.value)
      localStorage.setItem(ACTIVE_TAB_KEY, guid)
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
      localStorage.setItem(ACTIVE_TAB_KEY, next?.guid ?? '')
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

  async function updateItemIcon(guid: string, icon: string, iconUrl: string) {
    const app = store.value.apps[guid]
    if (!app)
      return
    app.icon = icon
    state.value[guid] = { ...state.value[guid], icon_url: iconUrl }
    await save()
    showToast('Icon updated')
  }

  async function updateItem(guid: string, fields: Partial<AppItem>) {
    const app = store.value.apps[guid]
    if (!app)
      return
    Object.assign(app, fields)
    await save()
    showToast('Item updated')
  }

  async function setAutoMinimize(enabled: boolean) {
    store.value.settings.auto_minimize = enabled
    await save()
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
    updateItemIcon,
    updateItem,
    setAutoMinimize,
    setAbsolutePaths,
    convertToAbsolute,
    convertToRelative,
  }
}
