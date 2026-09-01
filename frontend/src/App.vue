<script setup lang="ts">
import type { AppItem } from './api'
import { Menu, MenuButton, MenuItem, MenuItems, TransitionRoot } from '@headlessui/vue'
import { computed, ref } from 'vue'
import { OpenDirectory } from './api'
import AppDialog from './components/AppDialog.vue'
import ItemEditDialog from './components/ItemEditDialog.vue'
import LauncherRow from './components/LauncherRow.vue'
import TabBar from './components/TabBar.vue'
import { useConfirmDialog } from './composables/useConfirmDialog'
import { useModalDialog } from './composables/useModalDialog'
import { useStore } from './composables/useStore'
import { useTheme } from './composables/useTheme'
import { useToast } from './composables/useToast'
import { showError } from './utils'

const { activeTab, state, store, addFilesInto, addTab, renameItem, removeItem, setActiveTab, moveTab, renameTab, removeTab, updateItemIcon, updateItem, save, refresh, setAutoMinimize, setAbsolutePaths, convertToAbsolute, convertToRelative } = useStore()
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

function onItemSaved(guid: string, fields: { name: string, path: string, args: string, working_dir: string }) {
  updateItem(guid, fields).catch(showError)
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

function onOpenProgramDir() {
  OpenDirectory('').catch(showError)
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
              <MenuItem v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="onOpenProgramDir"
                >
                  <span class="inline-flex h-4 w-4 shrink-0 items-center justify-center text-gray-500 dark:text-gray-400">
                    📂
                  </span>
                  Open program directory
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
        @rename="openItemRename(row.item)" @details="openItemEdit(row.item)" @delete="onDeleteRequested(row.item)"
        @icondone="(icon, iconUrl) => onIconDone(icon, iconUrl, row.item)" @dragstart="onDragStart(index)"
        @dragover="onDragOver(index)" @drop="onDrop(index)" @dragend="onDragEnd"
      />
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
