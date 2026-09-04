<script setup lang="ts">
import type { Category } from '../composables/useStore'
import type { MenuEntry } from '../composables/itemMenu'
import { Ellipsis, PencilLine, Plus, Trash2 } from '@lucide/vue'
import { ref, watch } from 'vue'
import ItemMenu from './ItemMenu.vue'

const props = defineProps<{
  tabs: Category[]
  activeGuid: string
  // GUID of the item currently being dragged (null when dragging a tab).
  dragItemGuid: string | null
}>()

const emit = defineEmits<{
  'add': []
  'select': [guid: string]
  'rename': [guid: string, name: string]
  'remove': [guid: string, name: string]
  'reorder': [from: number, to: number]
  'item-drop': [guid: string, tabGuid: string, copy: boolean]
}>()

// Drag: highlight the source (opacity) and the insertion target (ring) while
// dragging — no live DOM reordering, so no flicker. The real reorder happens
// once on drop (animated via TransitionGroup move).
const dragIndex = ref<number | null>(null)
const fromIndex = ref<number | null>(null)
const overIndex = ref<number | null>(null)

// Item-over-tab drop targets (move by default, copy while Ctrl/Meta held).
const itemOverIndex = ref<number | null>(null)
const itemCopy = ref(false)


function tabMenuEntries(tab: Category): MenuEntry[] {
  return [
    { key: 'rename', icon: PencilLine, label: 'Rename', action: () => emit('rename', tab.guid, tab.name) },
    { key: 'delete', icon: Trash2, label: 'Delete', danger: true, action: () => emit('remove', tab.guid, tab.name) },
  ]
}

function onDragStart(index: number) {
  fromIndex.value = index
  dragIndex.value = index
  overIndex.value = null
}

function onDragOver(e: DragEvent, index: number) {
  e.preventDefault()
  // An item is being dragged over this tab -> show the move/copy target.
  if (props.dragItemGuid) {
    itemOverIndex.value = index
    itemCopy.value = e.ctrlKey || e.metaKey
    return
  }
  // A tab is being dragged -> reorder preview.
  if (dragIndex.value === null)
    return
  overIndex.value = index
}

function onDrop(e: DragEvent, index: number) {
  e.preventDefault()
  // Item dropped onto a tab.
  if (props.dragItemGuid) {
    const guid = e.dataTransfer?.getData('application/x-go-launcher-item') || props.dragItemGuid
    const tab = props.tabs[index]
    if (tab && guid)
      emit('item-drop', guid, tab.guid, e.ctrlKey || e.metaKey)
    resetItemOver()
    return
  }
  // Tab dropped onto another tab -> reorder.
  if (fromIndex.value !== null && overIndex.value !== null && fromIndex.value !== overIndex.value)
    emit('reorder', fromIndex.value, overIndex.value)
  resetDrag()
}

function onDragEnd() {
  resetDrag()
}

function resetDrag() {
  fromIndex.value = null
  dragIndex.value = null
  overIndex.value = null
}

function resetItemOver() {
  itemOverIndex.value = null
  itemCopy.value = false
}

// When the dragged item is released anywhere, clear the tab highlight.
watch(() => props.dragItemGuid, (v) => {
  if (!v)
    resetItemOver()
})
</script>

<template>
  <div
    class="flex items-center gap-1 border-b border-gray-300 bg-white px-2 py-1.5 dark:border-gray-700 dark:bg-gray-800"
  >
    <TransitionGroup name="list" tag="div" class="flex items-center gap-1">
      <div
        v-for="(tab, index) in tabs" :key="tab.guid" :draggable="true"
        class="group relative flex items-center gap-1 rounded px-2.5 py-1 select-none" :class="[
          tab.guid === activeGuid
            ? 'bg-blue-500 text-white'
            : 'text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700',
          dragIndex === index ? 'opacity-40' : '',
          overIndex === index && dragIndex !== null && overIndex !== dragIndex ? 'ring-2 ring-blue-400' : '',
          itemOverIndex === index && dragItemGuid ? (itemCopy ? 'ring-2 ring-green-500' : 'ring-2 ring-blue-400') : '',
        ]" :title="itemOverIndex === index && dragItemGuid ? (itemCopy ? 'Drop to copy item to this tab' : 'Drop to move item to this tab') : ''"
        @click="emit('select', tab.guid)" @dragstart="onDragStart(index)" @dragover="onDragOver($event, index)"
        @drop="onDrop($event, index)" @dragend="onDragEnd"
      >
        <span class="text-sm">{{ tab.name }}</span>
        <span
          v-if="itemOverIndex === index && itemCopy && dragItemGuid"
          class="pointer-events-none absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-green-500 text-[11px] font-bold leading-none text-white ring-2 ring-white dark:ring-gray-800"
        >+</span>

        <ItemMenu :entries="tabMenuEntries(tab)" :estimate="100" align="left" width-class="w-32" button-class="flex h-4 w-4 items-center justify-center rounded text-xs opacity-0 hover:bg-white/20 group-hover:opacity-100">
          <template #button>
            <Ellipsis class="h-3 w-3" />
          </template>
        </ItemMenu>
      </div>
    </TransitionGroup>

    <button
      class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
      title="Add Tab" @click="emit('add')"
    >
      <Plus class="h-4 w-4" />
    </button>

    <slot />
  </div>
</template>
