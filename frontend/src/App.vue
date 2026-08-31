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
import { showError } from './utils'

const { activeTab, store, addFilesInto, addTab, renameItem, removeItem, setActiveTab, moveTab, renameTab, removeTab, updateItemIcon, save } = useStore()
const { theme, setTheme } = useTheme()

const themeOptions = [
  { value: 'auto', label: 'Auto' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
] as const

const rows = computed<{ item: AppItem, index: number }[]>(() => {
  const tab = activeTab.value
  if (!tab)
    return []
  return tab.slots
    .map((slot, index) => ({ item: slot, index }))
    .filter(x => x.item !== null) as { item: AppItem, index: number }[]
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

function onIconDone(icon: string, item: AppItem) {
  updateItemIcon(item.guid, icon).catch(showError)
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
    tab.slots = list.map(x => x.item)
    save()
  }
  draggingIndex.value = null
  dragOverIndex.value = null
}

function onDragEnd() {
  draggingIndex.value = null
  dragOverIndex.value = null
}

function onAddFiles() {
  addFilesInto().catch(showError)
}
</script>

<template>
  <div class="flex h-screen flex-col bg-gray-100 text-sm text-gray-800 dark:bg-gray-900 dark:text-gray-100">
    <header class="flex items-center gap-2.5 border-b border-gray-300 bg-white px-3 py-2 dark:border-gray-700 dark:bg-gray-800">
      <button
        class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 cursor-pointer dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
        @click="onAddFiles"
      >
        Add Files
      </button>
      <span class="text-gray-500 dark:text-gray-400">Drop files anywhere to add them</span>

      <Menu as="div" class="relative ml-auto">
        <MenuButton class="rounded border border-gray-400 bg-white px-2 py-1 hover:bg-gray-200 cursor-pointer dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600">
          ⋯
        </MenuButton>
        <TransitionRoot
          enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
          enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in"
          leave-from="opacity-100 scale-100" leave-to="opacity-0 scale-95"
        >
          <MenuItems class="absolute right-0 z-10 mt-1 w-56 origin-top-right rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800">
            <MenuItem v-slot="{ active }">
              <button
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
                @click="store.settings.auto_minimize = !store.settings.auto_minimize; save()"
              >
                <span
                  class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border text-xs"
                  :class="store.settings.auto_minimize ? 'border-blue-500 bg-blue-500 text-white' : 'border-gray-400 bg-white dark:border-gray-500 dark:bg-gray-700'"
                >
                  <span v-if="store.settings.auto_minimize">✓</span>
                </span>
                Auto-minimize window
              </button>
            </MenuItem>
            <div class="my-1 border-t border-gray-200 dark:border-gray-700" />
            <p class="px-3 pb-1 pt-1.5 text-xs uppercase tracking-wide text-gray-400 dark:text-gray-500">
              Theme
            </p>
            <MenuItem v-for="opt in themeOptions" :key="opt.value" v-slot="{ active }">
              <button
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
                @click="setTheme(opt.value)"
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
    </header>

    <TabBar
      :tabs="store.tabs"
      :active-guid="store.active_tab_guid"
      @add="addTab().catch(showError)"
      @select="setActiveTab"
      @rename="openTabRename"
      @remove="onDeleteTabRequested"
      @reorder="moveTab"
    />

    <main class="flex-1 overflow-y-auto p-2">
      <div v-if="rows.length === 0" class="p-5 text-center text-gray-500 dark:text-gray-400">
        No files added yet. Click "Add Files" or drop files anywhere.{{ activeTab ? ` (tab: ${activeTab.name})` : '' }}
      </div>
      <LauncherRow
        v-for="(row, index) in rows" :key="row.item.guid" :item="row.item"
        :dragging="draggingIndex === index"
        :drag-over="dragOverIndex === index && draggingIndex !== null && draggingIndex !== index"
        @rename="openItemRename(row.item)" @details="openItemDetails(row.item)"
        @delete="onDeleteRequested(row.item)"
        @icondone="icon => onIconDone(icon, row.item)"
        @dragstart="onDragStart(index)" @dragover="onDragOver(index)" @drop="onDrop(index)" @dragend="onDragEnd"
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
  </div>
</template>
