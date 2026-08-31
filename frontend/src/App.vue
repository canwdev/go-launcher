<script setup lang="ts">
import { ref } from 'vue'
import { AddFiles } from './api'
import ContextMenu from './components/ContextMenu.vue'
import LauncherRow from './components/LauncherRow.vue'
import ModalDialog from './components/ModalDialog.vue'
import { useLauncher } from './composables/useLauncher'
import { showError } from './utils'

const { items } = useLauncher()

const menu = ref<{ index: number, x: number, y: number } | null>(null)
const modalOpen = ref(false)
const modalMode = ref<'rename' | 'details'>('rename')
const modalTitle = ref('Rename')
const activeId = ref(-1)

function onMenu(index: number, position: { x: number, y: number }) {
  menu.value = { index, ...position }
}

function closeMenu() {
  menu.value = null
}

function openModal(mode: 'rename' | 'details', index: number) {
  activeId.value = index
  modalMode.value = mode
  modalTitle.value = mode === 'rename' ? 'Rename' : 'Details'
  modalOpen.value = true
}

function closeModal() {
  modalOpen.value = false
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
        @menu="(position) => onMenu(index, position)"
      />
    </main>

    <ContextMenu
      v-if="menu"
      :index="menu.index"
      :x="menu.x"
      :y="menu.y"
      @close="closeMenu"
      @rename="openModal('rename', menu.index)"
      @details="openModal('details', menu.index)"
    />

    <ModalDialog
      :open="modalOpen"
      :index="activeId"
      :mode="modalMode"
      :title="modalTitle"
      @close="closeModal"
    />
  </div>
</template>
