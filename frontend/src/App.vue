<script setup lang="ts">
import type { Component } from 'vue'
import type { AppItem } from './api'
import type { Theme } from './composables/useTheme'
import { Menu, MenuButton, MenuItem, MenuItems, TransitionRoot } from '@headlessui/vue'
import { Check, Ellipsis, FolderOpen, Gamepad2, Images, Moon, Plus, RefreshCw, Sun, SunMoon } from '@lucide/vue'
import { computed, ref } from 'vue'
import { OpenDirectory } from './api'
import AppDialog from './components/AppDialog.vue'
import ItemEditDialog from './components/ItemEditDialog.vue'
import LauncherRow from './components/LauncherRow.vue'
import RuntimeEditDialog from './components/RuntimeEditDialog.vue'
import TabBar from './components/TabBar.vue'
import { useConfirmDialog } from './composables/useConfirmDialog'
import { useModalDialog } from './composables/useModalDialog'
import { useStore } from './composables/useStore'
import { useTheme } from './composables/useTheme'
import { useToast } from './composables/useToast'
import { showError } from './utils'

const { activeTab, state, store, addFilesInto, addTab, renameItem, removeItem, setActiveTab, moveTab, renameTab, removeTab, updateItemIcon, updateItem, batchUpdateIcons, save, refresh, setGameMode, setRuntimeMs, setAbsolutePaths, convertToAbsolute, convertToRelative } = useStore()
const { theme, setTheme } = useTheme()
const { toasts, showToast } = useToast()

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

const { open: confirmOpen, message: confirmMessage, request: requestConfirm, confirm: onConfirm, close: closeConfirm } = useConfirmDialog()

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

function onIconDone(icon: string, iconUrl: string, item: AppItem) {
  updateItemIcon(item.guid, icon, iconUrl).catch(showError)
}

function onDragStart(index: number) {
  const tab = activeTab.value
  if (!tab)
    return
  dragFromIndex.value = index
  draggingIndex.value = index
  dragOverIndex.value = null
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
      :tabs="store.categories" :active-guid="activeTab?.guid ?? ''" @add="addTab().catch(showError)"
      @select="setActiveTab" @rename="openTabRename" @remove="onDeleteTabRequested" @reorder="moveTab"
    >
      <div class="flex flex-1 items-center justify-end gap-1">
        <button
          class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          title="Add Files" @click="onAddFiles"
        >
          <Plus class="h-4 w-4" />
        </button>

        <button
          class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
          :title="`Theme: ${themeLabel}`" @click="cycleTheme"
        >
          <component :is="themeIcon" class="h-4 w-4" />
        </button>

        <Menu as="div" class="relative">
          <MenuButton
            class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
            title="Menu"
          >
            <Ellipsis class="h-4 w-4" />
          </MenuButton>
          <TransitionRoot
            enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in" leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <MenuItems
              class="absolute right-0 z-10 mt-1 w-56 origin-top-right rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
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
          </TransitionRoot>
        </Menu>
      </div>
    </TabBar>

    <main class="flex-1 overflow-y-auto p-2">
      <Transition name="fade" mode="out-in">
        <div :key="activeTab?.guid ?? 'empty'" class="min-h-full">
          <div v-if="rows.length === 0" class="p-5 text-center text-gray-500 dark:text-gray-400">
            No files added yet. Click "Add Files" or drop files anywhere.{{ activeTab ? ` (tab: ${activeTab.name})` : '' }}
          </div>
          <TransitionGroup name="list">
            <LauncherRow
              v-for="(row, index) in rows" :key="row.item.guid" :item="row.item"
              :icon-url="state[row.item.guid]?.icon_url ?? ''" :running="state[row.item.guid]?.running ?? false"
              :runtime-ms="state[row.item.guid]?.runtime_ms ?? 0" :dragging="draggingIndex === index"
              :drag-over="dragOverIndex === index && draggingIndex !== null && dragOverIndex !== draggingIndex"
              :game-mode="store.settings.game_mode"
              @rename="openItemRename(row.item)" @details="openItemEdit(row.item)" @delete="onDeleteRequested(row.item)"
              @refresh="refresh" @icondone="(icon, iconUrl) => onIconDone(icon, iconUrl, row.item)"
              @edit-runtime="onEditRuntime(row.item.guid)"
              @dragstart="onDragStart(index)" @dragover="onDragOver(index)" @drop="onDrop" @dragend="onDragEnd"
            />
          </TransitionGroup>
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
      :open="runtimeEditOpen" :runtime-ms="runtimeEditMs" @save="onRuntimeSaved" @close="runtimeEditOpen = false"
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
          Delete
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
