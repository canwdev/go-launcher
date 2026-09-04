<script setup lang="ts">
import type { Component } from 'vue'
import type { AppItem } from './api'
import type { Theme } from './composables/useTheme'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { Check, Ellipsis, FolderOpen, Gamepad2, Images, LayoutGrid, List, Moon, Plus, RefreshCw, Sun, SunMoon } from '@lucide/vue'
import { useStorage } from '@vueuse/core'
import { computed, ref } from 'vue'
import { OpenDirectory } from './api'
import AppDialog from './components/AppDialog.vue'
import GridItem from './components/GridItem.vue'
import ItemEditDialog from './components/ItemEditDialog.vue'
import LauncherRow from './components/LauncherRow.vue'
import RuntimeEditDialog from './components/RuntimeEditDialog.vue'
import TabBar from './components/TabBar.vue'
import { useAutoRuntime } from './composables/useAutoRuntime'
import { useConfirmDialog } from './composables/useConfirmDialog'
import { useManualTimer } from './composables/useManualTimer'
import { useMenuFlip } from './composables/useMenuFlip'
import { useModalDialog } from './composables/useModalDialog'
import { useStore } from './composables/useStore'
import { useTheme } from './composables/useTheme'
import { useToast } from './composables/useToast'
import { showError } from './utils'

const { activeTab, state, store, addFilesInto, addTab, renameItem, removeItem, setActiveTab, moveTab, renameTab, removeTab, updateItemIcon, updateItem, batchUpdateIcons, save, refresh, setGameMode, setRuntimeMs, setAbsolutePaths, convertToAbsolute, convertToRelative, duplicateItem, moveItemToTab, copyItemToTab, insertEmptyAt, deleteSlotAt, reorderSlots } = useStore()
const { theme, setTheme } = useTheme()
const { toasts, showToast } = useToast()
const autoRuntime = useAutoRuntime()
const timer = useManualTimer()

// 视图模式：list（列表） / grid（网格）。UI 偏好，走 localStorage。
const viewMode = useStorage<'list' | 'grid'>('launcher-view-mode', 'list')
function toggleViewMode() {
  viewMode.value = viewMode.value === 'grid' ? 'list' : 'grid'
}

// === Grid 视图拖拽状态 ===
const gridDrag = ref<{ from: number, isSlot: boolean } | null>(null)
const gridOver = ref<number | null>(null)
const gridCopy = ref(false)

// GUID of the item currently being dragged (null when dragging a tab or nothing).
const dragItemGuid = ref<string | null>(null)
const ITEM_DRAG_MIME = 'application/x-go-launcher-item'

function onGridDragStart(index: number, isSlot: boolean) {
  gridDrag.value = { from: index, isSlot }
  gridOver.value = null
  gridCopy.value = false
  // 同步到标签栏拖拽：记录被拖 item 的 guid（空槽无 guid）
  const slot = activeTab.value?.slots[index]
  dragItemGuid.value = (!isSlot && slot) ? slot : null
}
function onGridDragOver(index: number) {
  if (gridDrag.value)
    gridOver.value = index
}
function onGridDrop(index: number, ctrl: boolean) {
  const d = gridDrag.value
  if (d && gridOver.value !== null && d.from !== index) {
    if (ctrl && !d.isSlot) {
      gridCopy.value = true
      reorderSlots(d.from, index, true).catch(showError)
    }
    else {
      reorderSlots(d.from, index, false).catch(showError)
    }
  }
  gridDrag.value = null
  gridOver.value = null
  gridCopy.value = false
  dragItemGuid.value = null
}
function onGridDragEnd() {
  gridDrag.value = null
  gridOver.value = null
  gridCopy.value = false
  dragItemGuid.value = null
}

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
// 显示为 "Xm (+ Xm)" 的括号部分；未计时返回 0。
function rowLiveMs(guid: string): number {
  if (timer.isAutoTimer(guid))
    return timer.elapsedMs(guid)
  const st = state.value[guid]
  const baseline = st?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
  const live = autoRuntime.liveMs(baseline, st?.running ?? false, st?.start_at ?? 0)
  return live > baseline ? live - baseline : 0
}

const themeSequence: Theme[] = ['auto', 'light', 'dark']
const themeIcon = computed(() => theme.value === 'dark' ? Moon : theme.value === 'light' ? Sun : SunMoon)
const themeLabel = computed(() => theme.value[0].toUpperCase() + theme.value.slice(1))

function cycleTheme() {
  const idx = themeSequence.indexOf(theme.value)
  setTheme(themeSequence[(idx + 1) % themeSequence.length])
}

interface MenuEntry { key: string, divider?: boolean, toggle?: boolean, label?: string, checked?: () => boolean, onClick?: () => void, icon?: Component }
const appMenuItems: MenuEntry[] = [
  {
    key: 'game-mode',
    toggle: true,
    icon: Gamepad2,
    label: 'Game mode',
    checked: () => store.value.settings.game_mode,
    onClick: () => setGameMode(!store.value.settings.game_mode),
  },
  {
    key: 'absolute-paths',
    toggle: true,
    label: 'Abs path for new items',
    checked: () => store.value.settings.absolute_paths,
    onClick: () => setAbsolutePaths(!store.value.settings.absolute_paths),
  },
  { key: 'divider-2', divider: true },
  {
    key: 'refresh',
    icon: RefreshCw,
    label: 'Refresh',
    onClick: onRefresh,
  },
  {
    key: 'open-dir',
    icon: FolderOpen,
    label: 'Open program directory...',
    onClick: onOpenProgramDir,
  },
  { key: 'divider-1', divider: true },
  {
    key: 'to-absolute',
    label: 'Convert to absolute path',
    onClick: () => convertToAbsolute(),
  },
  {
    key: 'to-relative',
    label: 'Convert to relative path',
    onClick: () => convertToRelative(),
  },
  {
    key: 'batch-icons',
    icon: Images,
    label: 'Batch update icons',
    onClick: onBatchUpdateIcons,
  },
]

// Flip the main menu upward when there is not enough room below the button.
const { onMenuButtonClick, menuPosition } = useMenuFlip({ estimate: 260 })

const draggingIndex = ref<number | null>(null)
const dragFromIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

const rows = computed<{ item: AppItem, index: number }[]>(() => {
  const tab = activeTab.value
  if (!tab)
    return []
  return tab.slots
    .map((guid, index) => ({ item: guid ? store.value.apps[guid] : undefined, index }))
    .filter((x): x is { item: AppItem, index: number } => x.item != null)
})

type ModalTarget = { kind: 'item', guid: string } | { kind: 'tab', guid: string }
const modalTarget = ref<ModalTarget | null>(null)

function onSubmitModal(name: string) {
  const target = modalTarget.value
  if (!target)
    return
  if (target.kind === 'item') {
    renameItem(target.guid, name).catch(showError)
  }
  else {
    renameTab(target.guid, name).catch(showError)
  }
}

const { open: modalOpen, title: modalTitle, name: modalName, openRename: openModalRename, ok: onModalOk, close: closeModal } = useModalDialog(onSubmitModal)

const { open: confirmOpen, message: confirmMessage, request: requestConfirm, requestAsync, confirm: onConfirm, close: closeConfirm } = useConfirmDialog()

const editOpen = ref(false)
const editingItem = ref<AppItem | null>(null)

function openItemEdit(item: AppItem) {
  editingItem.value = item
  editOpen.value = true
}

function onItemSaved(guid: string, fields: { name: string, path: string, args: string, working_dir: string, icon: string }) {
  updateItem(guid, fields).catch(showError)
}

const runtimeEditOpen = ref(false)
const runtimeEditGuid = ref<string | null>(null)
const runtimeEditMs = ref(0)

function onEditRuntime(guid: string) {
  runtimeEditGuid.value = guid
  runtimeEditMs.value = state.value[guid]?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
  runtimeEditOpen.value = true
}

function onRuntimeSaved(minutes: number) {
  if (!runtimeEditGuid.value)
    return
  setRuntimeMs(runtimeEditGuid.value, minutes * 60000).catch(showError)
}

// 把计时累计 ms 加到该 item 当前 runtime 后写回后端（SetRuntimeMs 是覆盖写，需前端算总和）。
async function commitTimerMs(guid: string, ms: number) {
  if (ms <= 0)
    return
  const current = state.value[guid]?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
  await setRuntimeMs(guid, current + ms).catch(showError)
}

function onStartTimer(guid: string) {
  timer.start(guid)
  runtimeEditOpen.value = false
}

function onStopTimer(guid: string) {
  timer.stop(guid, async (ms) => {
    if (ms <= 0) {
      showToast('Timer stopped')
      return
    }
    await commitTimerMs(guid, ms)
    showToast('Runtime saved')
  })
}

// 启动成功后，若该 item 开启了 autoTimer，自动触发手动计时（不抢占/不重置其它计时）。
function onLaunched(guid: string) {
  if (timer.isAutoTimer(guid))
    timer.start(guid)
}

function openItemRename(item: AppItem) {
  modalTarget.value = { kind: 'item', guid: item.guid }
  openModalRename('Rename', item.name)
}

function openTabRename(guid: string, name: string) {
  modalTarget.value = { kind: 'tab', guid }
  openModalRename('Rename Tab', name)
}

function onDeleteRequested(item: AppItem) {
  requestConfirm(`Delete "${item.name}"?`, () => removeItem(item.guid).catch(showError))
}

function onDeleteTabRequested(guid: string, name: string) {
  requestConfirm(`Delete tab "${name}"?`, () => removeTab(guid).catch(showError))
}

// grid 视图：停止运行中的程序前需用户确认（英文文案）
function onConfirmStop(resolve: (ok: boolean) => void) {
  requestAsync('Stop this app?').then(resolve)
}

function onIconDone(icon: string, iconUrl: string, item: AppItem) {
  updateItemIcon(item.guid, icon, iconUrl).catch(showError)
}

function onDragStart(e: DragEvent, index: number) {
  const row = rows.value[index]
  if (!row)
    return
  dragItemGuid.value = row.item.guid
  dragFromIndex.value = index
  draggingIndex.value = index
  dragOverIndex.value = null
  e.dataTransfer?.setData(ITEM_DRAG_MIME, row.item.guid)
  if (e.dataTransfer)
    e.dataTransfer.effectAllowed = 'copyMove'
}

function onDragOver(index: number) {
  if (draggingIndex.value === null)
    return
  dragOverIndex.value = index
}

function onDrop() {
  const tab = activeTab.value
  if (tab && dragFromIndex.value !== null && dragOverIndex.value !== null && dragFromIndex.value !== dragOverIndex.value) {
    const slots = [...tab.slots]
    const [guid] = slots.splice(dragFromIndex.value, 1)
    slots.splice(dragOverIndex.value, 0, guid)
    tab.slots = slots
    save()
  }
  resetDrag()
}

function onDragEnd() {
  resetDrag()
}

function resetDrag() {
  dragFromIndex.value = null
  draggingIndex.value = null
  dragOverIndex.value = null
  dragItemGuid.value = null
}

// Drop an item onto a tab: copy (Ctrl held) or move.
function onItemDropOnTab(guid: string, tabGuid: string, copy: boolean) {
  resetDrag()
  // 若从网格视图拖来，同样清空网格拖拽状态
  gridDrag.value = null
  gridOver.value = null
  gridCopy.value = false
  if (copy)
    copyItemToTab(guid, tabGuid).catch(showError)
  else
    moveItemToTab(guid, tabGuid).catch(showError)
}

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
  addFilesInto().catch(showError)
}
</script>

<template>
  <div class="flex h-screen flex-col bg-gray-100 text-sm text-gray-800 dark:bg-gray-900 dark:text-gray-100">
    <TabBar
      :tabs="store.categories" :active-guid="activeTab?.guid ?? ''" :drag-item-guid="dragItemGuid"
      @add="addTab().catch(showError)"
      @select="setActiveTab" @rename="openTabRename" @remove="onDeleteTabRequested" @reorder="moveTab"
      @item-drop="onItemDropOnTab"
    >
      <div class="flex flex-1 items-center justify-end gap-1">
        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          title="Add Files" @click="onAddFiles"
        >
          <Plus class="h-4 w-4" />
        </button>

        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          :title="`Theme: ${themeLabel}`" @click="cycleTheme"
        >
          <component :is="themeIcon" class="h-4 w-4" />
        </button>

        <!-- 视图切换：网格 ⇄ 列表（放在全局菜单左侧） -->
        <button
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          :title="viewMode === 'grid' ? 'Switch to list view' : 'Switch to grid view'"
          @click="toggleViewMode"
        >
          <component :is="viewMode === 'grid' ? List : LayoutGrid" class="h-4 w-4" />
        </button>

        <Menu as="div" class="relative">
          <MenuButton
            class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
            title="Menu" @click="onMenuButtonClick"
          >
            <Ellipsis class="h-4 w-4" />
          </MenuButton>
          <Teleport to="body">
            <MenuItems
              class="w-56 overflow-y-auto rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
              :style="menuPosition('right')"
            >
              <template v-for="item in appMenuItems" :key="item.key">
                <div v-if="item.divider" class="my-1 border-t border-gray-200 dark:border-gray-700" />
                <MenuItem v-else v-slot="{ active }">
                  <button
                    class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
                    :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="item.onClick?.()"
                  >
                    <span v-if="item.icon" class="inline-flex h-4 w-4 shrink-0 items-center justify-center text-gray-500 dark:text-gray-400">
                      <component :is="item.icon" class="h-4 w-4" />
                    </span>
                    <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
                    <Check v-if="item.toggle && item.checked?.()" class="h-4 w-4 shrink-0 text-blue-500" />
                  </button>
                </MenuItem>
              </template>
            </MenuItems>
          </Teleport>
        </Menu>
      </div>
    </TabBar>

    <main class="flex-1 overflow-y-auto p-2">
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
              :baseline-ms="rowBaselineMs(row.item.guid)" :live-ms="rowLiveMs(row.item.guid)" :dragging="draggingIndex === index"
              :drag-over="dragOverIndex === index && draggingIndex !== null && dragOverIndex !== draggingIndex"
              :game-mode="store.settings.game_mode"
              :timer-active="timer.isActive(row.item.guid)" :timer-ms="timer.elapsedMs(row.item.guid)"
              :auto-timer="timer.isAutoTimer(row.item.guid)"
              @rename="openItemRename(row.item)" @details="openItemEdit(row.item)" @duplicate="duplicateItem(row.item.guid)" @delete="onDeleteRequested(row.item)"
              @refresh="refresh" @icondone="(icon, iconUrl) => onIconDone(icon, iconUrl, row.item)"
              @edit-runtime="onEditRuntime(row.item.guid)" @stop-timer="onStopTimer(row.item.guid)"
              @launched="onLaunched(row.item.guid)"
              @insert-empty="onInsertEmptyByGuid(row.item.guid)"
              @dragstart="(e: DragEvent) => onDragStart(e, index)" @dragover="onDragOver(index)" @drop="onDrop" @dragend="onDragEnd"
            />
          </TransitionGroup>

          <!-- 网格视图（含空槽；随窗口宽度自动变列数） -->
          <div v-else class="grid gap-2 pb-1" style="grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));">
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
              :dragging="gridDrag?.from === i"
              :drag-over="gridOver === i && gridDrag !== null && gridOver !== gridDrag.from"
              :drag-copy="gridCopy"
              @rename="slot && openItemRename(gridItem(slot)!)"
              @details="slot && openItemEdit(gridItem(slot)!)"
              @duplicate="slot && duplicateItem(slot)"
              @insert-empty="onInsertEmptyAt(i)"
              @delete-empty="slot === null && deleteSlotAt(i)"
              @delete="slot && onDeleteRequested(gridItem(slot)!)"
              @refresh="refresh"
              @icondone="(icon, iconUrl) => slot && onIconDone(icon, iconUrl, gridItem(slot)!)"
              @edit-runtime="slot && onEditRuntime(slot)"
              @stop-timer="slot && onStopTimer(slot)"
              @launched="slot && onLaunched(slot)"
              @grid-dragstart="onGridDragStart"
              @grid-dragover="onGridDragOver"
              @grid-drop="onGridDrop"
              @grid-dragend="onGridDragEnd"
              @confirm-stop="onConfirmStop"
            />
          </div>
        </div>
      </Transition>
    </main>

    <AppDialog :open="modalOpen" :title="modalTitle" @close="closeModal">
      <input
        v-model="modalName" type="text" autofocus
        class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        @keyup.enter="onModalOk"
      >
      <template #actions>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="onModalOk"
        >
          OK
        </button>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="closeModal"
        >
          Cancel
        </button>
      </template>
    </AppDialog>

    <ItemEditDialog :open="editOpen" :item="editingItem" @save="onItemSaved" @close="editOpen = false" />

    <RuntimeEditDialog
      :open="runtimeEditOpen" :runtime-ms="runtimeEditMs"
      :timer-active="timer.isActive(runtimeEditGuid ?? '')"
      :auto-timer="timer.isAutoTimer(runtimeEditGuid ?? '')"
      @save="onRuntimeSaved" @close="runtimeEditOpen = false" @start-timer="onStartTimer(runtimeEditGuid ?? '')"
      @auto-timer="(v: boolean) => timer.setAutoTimer(runtimeEditGuid ?? '', v)"
    />

    <AppDialog :open="confirmOpen" title="Confirm" @close="closeConfirm">
      <p class="m-0">
        {{ confirmMessage }}
      </p>
      <template #actions>
        <button
          class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
          @click="closeConfirm"
        >
          Cancel
        </button>
        <button
          class="rounded border border-red-600 bg-red-500 px-2.5 py-1 text-white hover:bg-red-600"
          @click="onConfirm"
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
  </div>
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
