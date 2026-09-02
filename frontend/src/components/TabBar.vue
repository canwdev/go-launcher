<script setup lang="ts">
import type { Category } from '../composables/useStore'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { Ellipsis, PencilLine, Plus, Trash2 } from '@lucide/vue'
import { ref, watch } from 'vue'
import { useMenuFlip } from '../composables/useMenuFlip'

const props = defineProps<{
  tabs: Category[]
  activeGuid: string
  // GUID of the item currently being dragged (null when dragging a tab).
  dragItemGuid: string | null
}>()

const emit = defineEmits<{
  add: []
  select: [guid: string]
  rename: [guid: string, name: string]
  remove: [guid: string, name: string]
  reorder: [from: number, to: number]
  itemDrop: [guid: string, tabGuid: string, copy: boolean]
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

// Flip the tab menu upward when there is not enough room below.
const { onMenuButtonClick, menuPosition } = useMenuFlip({ estimate: 100 })

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
      emit('itemDrop', guid, tab.guid, e.ctrlKey || e.metaKey)
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
    <TransitionGroup name="list" class="flex items-center gap-1">
      <div
        v-for="(tab, index) in tabs" :key="tab.guid" :draggable="true"
        class="group relative flex cursor-pointer items-center gap-1 rounded px-2.5 py-1 select-none" :class="[
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

        <Menu as="div" class="relative">
          <MenuButton
            class="flex h-4 w-4 items-center justify-center rounded text-xs opacity-0 hover:bg-white/20 group-hover:opacity-100"
            @click.stop="onMenuButtonClick"
          >
            <Ellipsis class="h-3 w-3" />
          </MenuButton>
          <Teleport to="body">
            <MenuItems
              class="w-32 overflow-y-auto rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
              :style="menuPosition('left')"
            >
              <MenuItem v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-700 dark:text-gray-200"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="emit('rename', tab.guid, tab.name)"
                >
                  <PencilLine class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
                  <span>Rename</span>
                </button>
              </MenuItem>
              <MenuItem v-slot="{ active }">
                <button
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-red-600 dark:text-red-400"
                  :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="emit('remove', tab.guid, tab.name)"
                >
                  <Trash2 class="h-4 w-4 shrink-0" />
                  <span>Delete</span>
                </button>
              </MenuItem>
            </MenuItems>
          </Teleport>
        </Menu>
      </div>
    </TransitionGroup>

    <button
      class="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
      title="Add Tab" @click="emit('add')"
    >
      <Plus class="h-4 w-4" />
    </button>

    <slot />
  </div>
</template>
