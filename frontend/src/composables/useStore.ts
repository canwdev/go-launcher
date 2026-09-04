import type { AppData, AppItem, AppStore, ItemState } from '../api'
import { useStorage } from '@vueuse/core'
import { onMounted, onUnmounted, ref } from 'vue'
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

  // 全局 busy 状态：耗时异步操作（批量图标/路径转换/添加文件）期间驱动 UI loading 提示
  const busy = ref(false)
  const busyMessage = ref('')

  async function runBusy<T>(message: string, fn: () => Promise<T>): Promise<T> {
    busy.value = true
    busyMessage.value = message
    try {
      return await fn()
    }
    finally {
      busy.value = false
      busyMessage.value = ''
    }
  }

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

  function defaultTitle(p: string): string {
    const norm = p.replace(/[\\/]+\$/, '')
    const idx = Math.max(norm.lastIndexOf('\\'), norm.lastIndexOf('/'))
    const base = idx >= 0 ? norm.slice(idx + 1) : norm
    const dot = base.lastIndexOf('.')
    return dot > 0 ? base.slice(0, dot) : base
  }

  async function createItem(fields: { name: string, path: string, args: string, working_dir: string, icon: string }) {
    // path 可为空：用于仅打开 Working directory 的目录型 item；name 为空才拦截
    const name = fields.name.trim() || defaultTitle(fields.path)
    if (!name)
      return
    ensureActive()
    if (!activeTab.value) {
      await addTab('Default')
      ensureActive()
    }
    if (!activeTab.value)
      return
    const guid = randomUUID()
    const item: AppItem = {
      guid,
      name,
      path: fields.path.trim(),
      args: fields.args.trim(),
      working_dir: fields.working_dir.trim(),
      icon: fields.icon.trim(),
      runtime_ms: 0,
    }
    store.value.apps[guid] = item
    activeTab.value.slots.push(guid)
    await save()
    showToast('Item created')
  }

  async function addFilesInto() {
    try {
      await runBusy('Adding files...', async () => {
        const res = await AddFiles()
        if (res.items?.length)
          await addItems(res.items, res.icons)
      })
    }
    catch (err) {
      showError(err)
    }
  }

  function pruneOrphans() {
    // 保留空槽：仅按非 null 的 guid 计算引用，不重写 slots（空槽 null 需持久化）
    const referenced = new Set(store.value.categories.flatMap(c => c.slots.filter((s): s is string => s != null)))
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

  // === Grid 视图专用：空槽与重排操作 ===

  /** 在 activeTab 的指定槽位之前插入一个空槽（null）。item 菜单调用=插到 item 前面 */
  async function insertEmptyAt(index: number) {
    const tab = activeTab.value
    if (!tab)
      return
    tab.slots.splice(Math.max(0, Math.min(index, tab.slots.length)), 0, null)
    await save()
  }

  /** 删除 activeTab 指定索引处的空槽（仅限 null 槽位） */
  async function deleteSlotAt(index: number) {
    const tab = activeTab.value
    if (!tab || index < 0 || index >= tab.slots.length || tab.slots[index] !== null)
      return
    tab.slots.splice(index, 1)
    await save()
  }

  /**
   * Grid 拖拽重排：把源槽（item 或空槽）移动到目标位置；copy=true 时复制 item 到目标位。
   * 列表视图的拖拽也复用它（slot index 语义，正确处理含空槽场景）。
   */
  async function reorderSlots(from: number, to: number, copy = false) {
    const tab = activeTab.value
    if (!tab)
      return
    if (from < 0 || from >= tab.slots.length || to < 0 || to >= tab.slots.length || from === to)
      return
    const src = tab.slots[from]
    if (copy) {
      if (src == null)
        return // 空槽不能复制
      const app = store.value.apps[src]
      if (!app)
        return
      const copyItem: AppItem = {
        ...app,
        guid: randomUUID(),
        name: `${app.name} (copy)`,
        runtime_ms: 0,
      }
      store.value.apps[copyItem.guid] = copyItem
      if (state.value[src]?.icon_url)
        state.value[copyItem.guid] = { ...state.value[copyItem.guid], icon_url: state.value[src].icon_url }
      tab.slots.splice(to, 0, copyItem.guid)
    }
    else {
      const [moved] = tab.slots.splice(from, 1)
      tab.slots.splice(to, 0, moved ?? null)
    }
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
    // 本地 Object.assign 已是权威数据，SaveData 直接持久化，无需再全量 refresh
    Object.assign(app, fields)
    await saveNow()
    showToast('Item updated')
  }

  async function batchUpdateIcons() {
    await runBusy('Updating icons...', async () => {
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
    })
  }

  async function setGameMode(enabled: boolean) {
    store.value.settings.game_mode = enabled
    await save()
  }

  async function setRuntimeMs(guid: string, ms: number) {
    const cleanMs = Math.max(0, Math.floor(ms))
    await SetRuntimeMs(guid, cleanMs)
    // SetRuntimeMs 后端不推送 state:updated，本地同步两处即可，避免全量 refresh
    const app = store.value.apps[guid]
    if (app)
      app.runtime_ms = cleanMs
    const st = state.value[guid]
    if (st)
      st.runtime_ms = cleanMs
  }

  async function setAbsolutePaths(enabled: boolean) {
    store.value.settings.absolute_paths = enabled
    await save()
  }

  async function convertToAbsolute() {
    try {
      await runBusy('Converting to absolute path...', async () => {
        await ConvertToAbsolute()
        await refresh()
      })
      showToast('Converted to absolute path')
    }
    catch (err) {
      showError(err)
    }
  }

  async function convertToRelative() {
    try {
      await runBusy('Converting to relative path...', async () => {
        await ConvertToRelative()
        await refresh()
      })
      showToast('Converted to relative path')
    }
    catch (err) {
      showError(err)
    }
  }

  function onDrop(_x: number, _y: number, paths: string[]) {
    AddPaths(paths)
      .then((res) => {
        if (res.items?.length)
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
    busy,
    busyMessage,
    refresh,
    save,
    setActiveTab,
    addTab,
    removeTab,
    renameTab,
    moveTab,
    addItems,
    createItem,
    addFilesInto,
    removeItem,
    renameItem,
    duplicateItem,
    moveItemToTab,
    copyItemToTab,
    insertEmptyAt,
    deleteSlotAt,
    reorderSlots,
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
