<script setup lang="ts">
import type { AppItem } from './api'
import { Menu, MenuButton, MenuItem, MenuItems, TransitionRoot } from '@headlessui/vue'
import { computed, ref } from 'vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import LauncherRow from './components/LauncherRow.vue'
import ModalDialog from './components/ModalDialog.vue'
import TabBar from './components/TabBar.vue'
import { useStore } from './composables/useStore'
import { useTheme } from './composables/useTheme'
import { useToast } from './composables/useToast'
import { showError } from './utils'

const { activeTab, state, store, addFilesInto, addTab, renameItem, removeItem, setActiveTab, moveTab, renameTab, removeTab, updateItemIcon, save, refresh, setAutoMinimize, setAbsolutePaths, convertToAbsolute, convertToRelative } = useStore()
const { theme, setTheme } = useTheme()
const { toasts, showToast } = useToast()

const themeOptions = [
  { value: 'auto', label: 'Auto' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
] as const

const pathMenuItems = [
  {
    key: 'auto-minimize',
    toggle: true,
    label: 'Auto-minimize window',
    checked: () => store.value.settings.auto_minimize,
    onClick: () => setAutoMinimize(!store.value.settings.auto_minimize),
  },
  {
    key: 'absolute-paths',
    toggle: true,
    label: 'New items using absolute path',
    checked: () => store.value.settings.absolute_paths,
    onClick: () => setAbsolutePaths(!store.value.settings.absolute_paths),
  },
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
]

const rows = computed<{ item: AppItem, index: number }[]>(() => {
  const tab = activeTab.value
  if (!tab)
    return []
  return tab.slots
    .map((guid, index) => ({ item: guid ? store.value.apps[guid] : undefined, index }))
    .filter((x): x is { item: AppItem, index: number } => x.item != null)
})

const draggingIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

const modalOpen = ref(false)
const modalMode = ref<'rename' | 'details'>('rename')
const modalTitle = ref('')
const modalInitialName = ref('')
const modalDetails = ref('')
type ModalTarget = { kind: 'item', guid: string } | { kind: 'tab', guid: string }
const modalTarget = ref<ModalTarget | null>(null)

const confirmOpen = ref(false)
const confirmMessage = ref('')
const confirmAction = ref<null | (() => void)>(null)

function openItemDetails(item: AppItem) {
  modalMode.value = 'details'
  modalTitle.value = item.name
  modalDetails.value = JSON.stringify(item, null, 2)
  modalOpen.value = true
}

function openItemRename(item: AppItem) {
  modalMode.value = 'rename'
  modalTitle.value = 'Rename'
  modalInitialName.value = item.name
  modalTarget.value = { kind: 'item', guid: item.guid }
  modalOpen.value = true
}

function openTabRename(guid: string, name: string) {
  modalMode.value = 'rename'
  modalTitle.value = 'Rename Tab'
  modalInitialName.value = name
  modalTarget.value = { kind: 'tab', guid }
  modalOpen.value = true
}

function onModalOk(name: string) {
  if (!name)
    return
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

function onDeleteRequested(item: AppItem) {
  confirmMessage.value = `Delete "${item.name}"?`
  confirmAction.value = () => removeItem(item.guid).catch(showError)
  confirmOpen.value = true
}

function onDeleteTabRequested(guid: string, name: string) {
  confirmMessage.value = `Delete tab "${name}"?`
  confirmAction.value = () => removeTab(guid).catch(showError)
  confirmOpen.value = true
}

function onConfirm() {
  confirmOpen.value = false
  confirmAction.value?.()
  confirmAction.value = null
}

function onIconDone(icon: string, iconUrl: string, item: AppItem) {
  updateItemIcon(item.guid, icon, iconUrl).catch(showError)
}

function onDragStart(index: number) {
  draggingIndex.value = index
  dragOverIndex.value = null
}

function onDragOver(index: number) {
  dragOverIndex.value = index
}

function onDrop(target: number) {
  if (draggingIndex.value !== null && draggingIndex.value !== target) {
    const tab = activeTab.value
    if (!tab)
      return
    const list = rows.value
    const [moved] = list.splice(draggingIndex.value, 1)
    list.splice(target, 0, moved)
    tab.slots = list.map(x => x.item.guid)
    save()
  }
  draggingIndex.value = null
  dragOverIndex.value = null
}

function onDragEnd() {
  draggingIndex.value = null
  dragOverIndex.value = null
}

async function onRefresh() {
  await refresh()
  showToast('Data refreshed')
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
          ＋
        </button>

        <Menu as="div" class="relative">
          <MenuButton
            class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
            title="Menu"
          >
            ⋯
          </MenuButton>
          <TransitionRoot
            enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in" leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <MenuItems
              class="absolute right-0 z-10 mt-1 w-56 origin-top-right rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
            >
              <MenuItem v-for="item in pathMenuItems" :key="item.key" v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="item.onClick"
                >
                  <span
                    v-if="item.toggle" class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border text-xs"
                    :class="item.checked() ? 'border-blue-500 bg-blue-500 text-white' : 'border-gray-400 bg-white dark:border-gray-500 dark:bg-gray-700'"
                  >
                    <span v-if="item.checked()">✓</span>
                  </span>
                  {{ item.label }}
                </button>
              </MenuItem>
              <div class="my-1 border-t border-gray-200 dark:border-gray-700" />
              <MenuItem v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="onRefresh"
                >
                  <span class="inline-flex h-4 w-4 shrink-0 items-center justify-center text-gray-500 dark:text-gray-400">
                    ⟳
                  </span>
                  Refresh
                </button>
              </MenuItem>
              <div class="my-1 border-t border-gray-200 dark:border-gray-700" />
              <p class="px-3 pb-1 pt-1.5 text-xs uppercase tracking-wide text-gray-400 dark:text-gray-500">
                Theme
              </p>
              <MenuItem v-for="opt in themeOptions" :key="opt.value" v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="setTheme(opt.value)"
                >
                  <span
                    class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full border"
                    :class="theme === opt.value ? 'border-blue-500' : 'border-gray-400 dark:border-gray-500'"
                  >
                    <span v-if="theme === opt.value" class="h-2 w-2 rounded-full bg-blue-500" />
                  </span>
                  {{ opt.label }}
                </button>
              </MenuItem>
            </MenuItems>
          </TransitionRoot>
        </Menu>
      </div>
    </TabBar>

    <main class="flex-1 overflow-y-auto p-2">
      <div v-if="rows.length === 0" class="p-5 text-center text-gray-500 dark:text-gray-400">
        No files added yet. Click "Add Files" or drop files anywhere.{{ activeTab ? ` (tab: ${activeTab.name})` : '' }}
      </div>
      <LauncherRow
        v-for="(row, index) in rows" :key="row.item.guid" :item="row.item"
        :icon-url="state[row.item.guid]?.icon_url ?? ''" :running="state[row.item.guid]?.running ?? false"
        :runtime-ms="state[row.item.guid]?.runtime_ms ?? 0" :dragging="draggingIndex === index"
        :drag-over="dragOverIndex === index && draggingIndex !== null && draggingIndex !== index"
        @rename="openItemRename(row.item)" @details="openItemDetails(row.item)" @delete="onDeleteRequested(row.item)"
        @icondone="(icon, iconUrl) => onIconDone(icon, iconUrl, row.item)" @dragstart="onDragStart(index)"
        @dragover="onDragOver(index)" @drop="onDrop(index)" @dragend="onDragEnd"
      />
    </main>

    <ModalDialog
      :open="modalOpen" :mode="modalMode" :title="modalTitle" :initial-name="modalInitialName"
      :details-text="modalDetails" @ok="onModalOk" @close="modalOpen = false"
    />

    <ConfirmDialog
      :open="confirmOpen" title="Confirm" :message="confirmMessage" @confirm="onConfirm"
      @close="confirmOpen = false"
    />

    <div class="pointer-events-none fixed bottom-3 left-1/2 z-50 flex -translate-x-1/2 flex-col items-center gap-2">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts" :key="toast.id"
          class="rounded border px-3 py-1.5 text-sm shadow-md"
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
