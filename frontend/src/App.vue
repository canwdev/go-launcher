<script setup lang="ts">
import {
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  TransitionRoot,
} from '@headlessui/vue'
import { ref } from 'vue'
import { AddFiles, MoveItem, RemoveItem } from './api'
import ConfirmDialog from './components/ConfirmDialog.vue'
import LauncherRow from './components/LauncherRow.vue'
import ModalDialog from './components/ModalDialog.vue'
import { useAutoMinimize } from './composables/useAutoMinimize'
import { useLauncher } from './composables/useLauncher'
import { showError } from './utils'

const { items } = useLauncher()
const { autoMinimize, toggle: toggleAutoMinimize } = useAutoMinimize()

const modalOpen = ref(false)
const modalMode = ref<'rename' | 'details'>('rename')
const modalTitle = ref('Rename')
const modalName = ref('')
const activeId = ref(-1)

const confirmOpen = ref(false)
const confirmMessage = ref('')
const deleteId = ref(-1)

const draggingIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)

function openModal(mode: 'rename' | 'details', index: number) {
  activeId.value = index
  modalMode.value = mode
  modalTitle.value = mode === 'rename' ? 'Rename' : 'Details'
  if (mode === 'rename')
    modalName.value = items.value[index]?.title ?? ''
  modalOpen.value = true
}

function closeModal() {
  modalOpen.value = false
}

function onDeleteRequested(index: number) {
  deleteId.value = index
  confirmMessage.value = `Delete "${items.value[index]?.title ?? ''}"?`
  confirmOpen.value = true
}

function onConfirmDelete() {
  confirmOpen.value = false
  RemoveItem(deleteId.value).catch(showError)
}

function onDragStart(index: number) {
  draggingIndex.value = index
  dragOverIndex.value = null
}

function onDragOver(index: number) {
  dragOverIndex.value = index
}

function onDrop(target: number) {
  if (draggingIndex.value !== null && draggingIndex.value !== target)
    MoveItem(draggingIndex.value, target).catch(showError)
  draggingIndex.value = null
  dragOverIndex.value = null
}

function onDragEnd() {
  draggingIndex.value = null
  dragOverIndex.value = null
}

function onAddFiles() {
  AddFiles().catch(showError)
}
</script>

<template>
  <div class="flex h-screen flex-col bg-gray-100 text-sm text-gray-800">
    <header class="flex items-center gap-2.5 border-b border-gray-300 bg-white px-3 py-2">
      <button
        class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
        @click="onAddFiles"
      >
        Add Files
      </button>
      <span class="text-gray-500">Drop files anywhere to add them</span>

      <Menu as="div" class="relative ml-auto">
        <MenuButton class="rounded border border-gray-400 bg-white px-2 py-1 hover:bg-gray-200">
          ⚙
        </MenuButton>
        <TransitionRoot
          enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
          enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in" leave-from="opacity-100 scale-100"
          leave-to="opacity-0 scale-95"
        >
          <MenuItems
            class="absolute right-0 z-10 mt-1 w-56 origin-top-right rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none"
          >
            <MenuItem v-slot="{ active }">
              <button
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
                @click="toggleAutoMinimize"
              >
                <span
                  class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border text-xs"
                  :class="autoMinimize ? 'border-blue-500 bg-blue-500 text-white' : 'border-gray-400 bg-white'"
                >
                  <span v-if="autoMinimize">✓</span>
                </span>
                Auto-minimize window
              </button>
            </MenuItem>
          </MenuItems>
        </TransitionRoot>
      </Menu>
    </header>

    <main class="flex-1 overflow-y-auto p-2">
      <div
        v-if="items.length === 0"
        class="p-5 text-center text-gray-500"
      >
        No files added yet. Click "Add Files" or drop files anywhere.
      </div>
      <LauncherRow
        v-for="(item, index) in items"
        :key="index"
        :item="item"
        :index="index"
        :dragging="draggingIndex === index"
        :drag-over="dragOverIndex === index && draggingIndex !== null && draggingIndex !== index"
        @rename="openModal('rename', index)"
        @details="openModal('details', index)"
        @delete="onDeleteRequested(index)"
        @dragstart="onDragStart(index)"
        @dragover="onDragOver(index)"
        @drop="onDrop(index)"
        @dragend="onDragEnd"
      />
    </main>

    <ModalDialog
      :open="modalOpen"
      :index="activeId"
      :mode="modalMode"
      :title="modalTitle"
      :initial-name="modalName"
      @close="closeModal"
    />

    <ConfirmDialog
      :open="confirmOpen"
      title="Confirm Delete"
      :message="confirmMessage"
      @confirm="onConfirmDelete"
      @close="confirmOpen = false"
    />
  </div>
</template>
