<script setup lang="ts">
import type { AppItem } from './api'
import type { Theme } from './composables/useTheme'
import { Ellipsis, LayoutGrid, List, LoaderCircle, Moon, Plus, Search, Sun, SunMoon } from '@lucide/vue'
import { useStorage } from '@vueuse/core'
import { computed, reactive, ref } from 'vue'
import { OpenDirectory } from './api'
import AppDialog from './components/AppDialog.vue'
import GridItem from './components/GridItem.vue'
import ItemEditDialog from './components/ItemEditDialog.vue'
import ItemMenu from './components/ItemMenu.vue'
import LauncherRow from './components/LauncherRow.vue'
import RuntimeEditDialog from './components/RuntimeEditDialog.vue'
import SearchOverlay from './components/SearchOverlay.vue'
import TabBar from './components/TabBar.vue'
import { useAutoRuntime } from './composables/useAutoRuntime'
import { useDialogs } from './composables/useDialogs'
import { useGridViewDrag } from './composables/useGridViewDrag'
import { useListViewDrag } from './composables/useListViewDrag'
import { useManualTimer } from './composables/useManualTimer'
import { useSearch } from './composables/useSearch'
import { useStore } from './composables/useStore'
import { useTheme } from './composables/useTheme'
import { showToast, useToast } from './composables/useToast'
import { buildAddMenu, buildAppMenu } from './menuConfig'
import { showError } from './utils'

const storeApi = useStore()
const {
  store,
  state,
  activeTab,
  busy,
  busyMessage,
  refresh,
  setGameMode,
  setAbsolutePaths,
  duplicateItem,
  moveItemToTab,
  copyItemToTab,
  insertEmptyAt,
  deleteSlotAt,
  reorderSlots,
  convertToAbsolute,
  convertToRelative,
  batchUpdateIcons,
  updateItemIcon,
} = storeApi
const { theme, setTheme } = useTheme()
const { toasts } = useToast()
const autoRuntime = useAutoRuntime()
const timer = useManualTimer()
// reactive 包裹：composable 返回的 ref 属性在模板中自动解包（嵌套 ref 不会默认解包）
const dialogs = reactive(useDialogs(storeApi, timer))
const { searchOpen, searchOnLaunch } = useSearch({ setActiveTab: storeApi.setActiveTab, timer })

// 视图模式：list（列表） / grid（网格）。UI 偏好，走 localStorage。
const viewMode = useStorage<'list' | 'grid'>('launcher-view-mode', 'list')

function toggleViewMode() {
  viewMode.value = viewMode.value === 'grid' ? 'list' : 'grid'
}

// === 拖拽状态（列表 + 网格共用 dragItemGuid，供 TabBar 显示 item-over-tab 目标） ===
const dragItemGuid = ref<string | null>(null)

const gridDrag = reactive(useGridViewDrag({
  dragItemGuid,
  getTab: () => activeTab.value,
  reorderSlots,
}))

// 列表视图渲染行：只含非空 item；index 为真实 slot index（slots 可能含 null 空槽）
const rows = computed<{ item: AppItem, index: number }[]>(() => {
  const tab = activeTab.value
  if (!tab)
    return []
  return tab.slots
    .map((guid, index) => ({ item: guid ? store.value.apps[guid] : undefined, index }))
    .filter((x): x is { item: AppItem, index: number } => x.item != null)
})

const listDrag = reactive(useListViewDrag({
  dragItemGuid,
  rows,
  getTab: () => activeTab.value,
  reorderSlots,
}))

// grid 渲染辅助：slot 转 item
function gridItem(slot: string | null): AppItem | null {
  return slot ? store.value.apps[slot] ?? null : null
}
function gridKey(slot: string | null, i: number): string {
  return slot ? `g-${slot}` : `s-${i}`
}
function onInsertEmptyAt(index: number) {
  insertEmptyAt(index).catch(showError)
}
function onInsertEmptyByGuid(guid: string) {
  const tab = activeTab.value
  if (!tab)
    return
  const idx = tab.slots.indexOf(guid)
  if (idx >= 0)
    insertEmptyAt(idx).catch(showError)
}

// 该 item 应展示的累计用时 baseline（ms），不含本次会话增量。
// autoTimer 项不触发自动计时：启动后增量完全走手动计时。
function rowBaselineMs(guid: string): number {
  const st = state.value[guid]
  return st?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
}

// 本次会话计时增量（ms）：autoTimer 项=手动计时 elapsed；普通项=自动计时 live 增量。
function rowLiveMs(guid: string): number {
  if (timer.isAutoTimer(guid))
    return timer.elapsedMs(guid)
  const st = state.value[guid]
  const baseline = st?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
  const live = autoRuntime.liveMs(baseline, st?.running ?? false, st?.start_at ?? 0)
  return live > baseline ? live - baseline : 0
}

// === 主题循环 ===
const themeSequence: Theme[] = ['auto', 'light', 'dark']
const themeIcon = computed(() => theme.value === 'dark' ? Moon : theme.value === 'light' ? Sun : SunMoon)
const themeLabel = computed(() => theme.value[0].toUpperCase() + theme.value.slice(1))

function cycleTheme() {
  const idx = themeSequence.indexOf(theme.value)
  setTheme(themeSequence[(idx + 1) % themeSequence.length])
}

// === 全局操作（供 appMenu / addMenu 回调） ===
async function onRefresh() {
  await refresh()
  showToast('Data refreshed')
}
function onOpenProgramDir() {
  OpenDirectory('').catch(showError)
}
function onBatchUpdateIcons() {
  batchUpdateIcons().catch(showError)
}
function onAddFiles() {
  storeApi.addFilesInto()
}

const appMenuItems = buildAppMenu({
  getGameMode: () => store.value.settings.game_mode,
  getAbsolutePaths: () => store.value.settings.absolute_paths,
  toggleGameMode: () => setGameMode(!store.value.settings.game_mode),
  toggleAbsolutePaths: () => setAbsolutePaths(!store.value.settings.absolute_paths),
  onRefresh,
  onOpenProgramDir,
  onConvertAbsolute: () => convertToAbsolute(),
  onConvertRelative: () => convertToRelative(),
  onBatchUpdateIcons,
})

const addMenuEntries = buildAddMenu({ onAddFiles, onCreate: dialogs.openCreateItem })

// item 图标更新完成回调
function onIconDone(icon: string, iconUrl: string, item: AppItem) {
  updateItemIcon(item.guid, icon, iconUrl).catch(showError)
}

// Drop an item onto a tab: copy (Ctrl held) or move. 清空两视图的拖拽状态。
function onItemDropOnTab(guid: string, tabGuid: string, copy: boolean) {
  listDrag.reset()
  gridDrag.reset()
  if (copy)
    copyItemToTab(guid, tabGuid).catch(showError)
  else
    moveItemToTab(guid, tabGuid).catch(showError)
}
</script>

<template>
  <div class="flex h-screen flex-col bg-gray-100 text-sm text-gray-800 dark:bg-gray-900 dark:text-gray-100">
    <TabBar
      :tabs="store.categories" :active-guid="activeTab?.guid ?? ''" :drag-item-guid="dragItemGuid"
      @add="storeApi.addTab().catch(showError)"
      @select="storeApi.setActiveTab" @rename="dialogs.openTabRename" @remove="dialogs.onDeleteTabRequested" @reorder="storeApi.moveTab"
      @item-drop="onItemDropOnTab"
    >
      <div class="flex flex-1 items-center justify-end gap-1">
        <ItemMenu :entries="addMenuEntries" width-class="w-40" button-class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700">
          <template #button>
            <Plus class="h-4 w-4" />
          </template>
        </ItemMenu>

        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          :title="`Theme: ${themeLabel}`" @click="cycleTheme"
        >
          <component :is="themeIcon" class="h-4 w-4" />
        </button>

        <!-- 视图切换：网格 ⇄ 列表 -->
        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          :title="viewMode === 'grid' ? 'Switch to list view' : 'Switch to grid view'"
          @click="toggleViewMode"
        >
          <component :is="viewMode === 'grid' ? List : LayoutGrid" class="h-4 w-4" />
        </button>

        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          title="Search" @click="searchOpen = true"
        >
          <Search class="h-4 w-4" />
        </button>

        <ItemMenu :entries="appMenuItems" :estimate="260" width-class="w-56" button-class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700">
          <template #button>
            <Ellipsis class="h-4 w-4" />
          </template>
        </ItemMenu>
      </div>
    </TabBar>

    <main class="flex-1 overflow-y-auto">
      <Transition name="fade" mode="out-in">
        <div :key="activeTab?.guid ?? 'empty'" class="min-h-full">
          <div v-if="viewMode === 'list' && rows.length === 0" class="p-5 text-center text-gray-500 dark:text-gray-400">
            No files added yet. Click "Add Files" or drop files anywhere.{{ activeTab ? ` (tab: ${activeTab.name})` : '' }}
          </div>
          <div v-else-if="viewMode === 'grid' && (!activeTab || activeTab.slots.length === 0)" class="p-5 text-center text-gray-500 dark:text-gray-400">
            No files added yet. Click "Add Files" or drop files anywhere.{{ activeTab ? ` (tab: ${activeTab.name})` : '' }}
          </div>

          <!-- 列表视图 -->
          <TransitionGroup v-if="viewMode === 'list'" name="list">
            <LauncherRow
              v-for="(row, index) in rows" :key="row.item.guid" :item="row.item"
              :icon-url="state[row.item.guid]?.icon_url ?? ''" :running="state[row.item.guid]?.running ?? false"
              :baseline-ms="rowBaselineMs(row.item.guid)" :live-ms="rowLiveMs(row.item.guid)" :dragging="listDrag.draggingSlot === row.index"
              :drag-over="listDrag.dragOverIndex === row.index && listDrag.draggingSlot !== null && listDrag.dragOverIndex !== listDrag.draggingSlot"
              :game-mode="store.settings.game_mode"
              :timer-active="timer.isActive(row.item.guid)" :timer-ms="timer.elapsedMs(row.item.guid)"
              :auto-timer="timer.isAutoTimer(row.item.guid)"
              @rename="dialogs.openItemRename(row.item)" @details="dialogs.openItemEdit(row.item)" @duplicate="duplicateItem(row.item.guid)" @delete="dialogs.onDeleteRequested(row.item)"
              @refresh="refresh" @icondone="(icon, iconUrl) => onIconDone(icon, iconUrl, row.item)"
              @edit-runtime="dialogs.onEditRuntime(row.item.guid)" @stop-timer="dialogs.onStopTimer(row.item.guid)"
              @launched="dialogs.onLaunched(row.item.guid)"
              @insert-empty="onInsertEmptyByGuid(row.item.guid)"
              @dragstart="(e: DragEvent) => listDrag.onDragStart(e, index)" @dragover="listDrag.onDragOver(index)" @drop="listDrag.onDrop($event)" @dragend="listDrag.onDragEnd"
            />
          </TransitionGroup>

          <!-- 网格视图（含空槽；随窗口宽度自动变列数） -->
          <div v-else class="grid gap-0" style="grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));">
            <GridItem
              v-for="(slot, i) in activeTab?.slots ?? []" :key="gridKey(slot, i)"
              :item="gridItem(slot)" :slot-index="i"
              :icon-url="slot ? state[slot]?.icon_url ?? '' : ''"
              :running="slot ? state[slot]?.running ?? false : false"
              :baseline-ms="slot ? rowBaselineMs(slot) : 0" :live-ms="slot ? rowLiveMs(slot) : 0"
              :game-mode="store.settings.game_mode"
              :timer-active="slot ? timer.isActive(slot) : false"
              :timer-ms="slot ? timer.elapsedMs(slot) : 0"
              :auto-timer="slot ? timer.isAutoTimer(slot) : false"
              :dragging="gridDrag.gridDrag?.from === i"
              :drag-over="gridDrag.gridOver === i && gridDrag.gridDrag !== null && gridDrag.gridOver !== gridDrag.gridDrag.from"
              :drag-copy="gridDrag.gridCopy"
              @rename="slot && dialogs.openItemRename(gridItem(slot)!)"
              @details="slot && dialogs.openItemEdit(gridItem(slot)!)"
              @duplicate="slot && duplicateItem(slot)"
              @insert-empty="onInsertEmptyAt(i)"
              @delete-empty="slot === null && deleteSlotAt(i)"
              @delete="slot && dialogs.onDeleteRequested(gridItem(slot)!)"
              @refresh="refresh"
              @icondone="(icon, iconUrl) => slot && onIconDone(icon, iconUrl, gridItem(slot)!)"
              @edit-runtime="slot && dialogs.onEditRuntime(slot)"
              @stop-timer="slot && dialogs.onStopTimer(slot)"
              @launched="slot && dialogs.onLaunched(slot)"
              @grid-dragstart="gridDrag.onGridDragStart"
              @grid-dragover="gridDrag.onGridDragOver"
              @grid-drop="gridDrag.onGridDrop"
              @grid-dragend="gridDrag.onGridDragEnd"
              @confirm-stop="dialogs.onConfirmStop"
            />
          </div>
        </div>
      </Transition>
    </main>

    <AppDialog :open="dialogs.modalOpen" :title="dialogs.modalTitle" @close="dialogs.closeModal">
      <input
        v-model="dialogs.modalName" type="text" autofocus
        class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        @keyup.enter="dialogs.onModalOk"
      >
      <template #actions>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="dialogs.onModalOk"
        >
          OK
        </button>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="dialogs.closeModal"
        >
          Cancel
        </button>
      </template>
    </AppDialog>

    <ItemEditDialog :open="dialogs.editOpen" :item="dialogs.editingItem" :creating="dialogs.editCreating" @save="dialogs.onItemSaved" @close="dialogs.editOpen = false" />

    <RuntimeEditDialog
      :open="dialogs.runtimeEditOpen" :runtime-ms="dialogs.runtimeEditMs"
      :timer-active="timer.isActive(dialogs.runtimeEditGuid ?? '')"
      :auto-timer="timer.isAutoTimer(dialogs.runtimeEditGuid ?? '')"
      @save="dialogs.onRuntimeSaved" @close="dialogs.runtimeEditOpen = false" @start-timer="dialogs.onStartTimer(dialogs.runtimeEditGuid ?? '')"
      @auto-timer="(v: boolean) => timer.setAutoTimer(dialogs.runtimeEditGuid ?? '', v)"
    />

    <AppDialog :open="dialogs.confirmOpen" title="Confirm" @close="dialogs.closeConfirm">
      <p class="m-0">
        {{ dialogs.confirmMessage }}
      </p>
      <template #actions>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="dialogs.closeConfirm"
        >
          Cancel
        </button>
        <button
          class="rounded border border-red-600 bg-red-500 px-2.5 py-1 text-white hover:bg-red-600"
          @click="dialogs.onConfirm"
        >
          OK
        </button>
      </template>
    </AppDialog>

    <div class="pointer-events-none fixed bottom-3 left-1/2 z-50 flex -translate-x-1/2 flex-col items-center gap-2">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts" :key="toast.id" class="rounded border px-3 py-1.5 text-sm shadow-md"
          :class="toast.type === 'error' ? 'border-red-400 bg-red-600 text-white' : 'border-green-600 bg-green-500 text-white'"
        >
          {{ toast.message }}
        </div>
      </TransitionGroup>
    </div>

    <!-- 耗时操作 loading 指示（批量图标 / 路径转换 / 添加文件） -->
    <div
      v-if="busy"
      class="pointer-events-none fixed bottom-12 left-1/2 z-40 flex -translate-x-1/2 items-center gap-2 rounded-full bg-gray-800/90 px-3 py-1.5 text-xs text-white shadow-lg"
    >
      <LoaderCircle class="h-3.5 w-3.5 animate-spin" />
      {{ busyMessage }}
    </div>
  </div>
  <SearchOverlay v-model:open="searchOpen" :store="store" :state="state" :on-launch="searchOnLaunch" />
</template>

<style scoped>
.toast-enter-active {
  animation: toast-in 0.2s ease-out;
}

.toast-leave-active {
  animation: toast-out 0.15s ease-in;
}

.toast-move {
  transition: transform 0.2s ease;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateY(0);
  }

  to {
    opacity: 0;
    transform: translateY(8px);
  }
}
</style>
