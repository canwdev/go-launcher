<script setup lang="ts">
import type { Category } from '../composables/useStore'
import { Menu, MenuButton, MenuItem, MenuItems, TransitionRoot } from '@headlessui/vue'
import { Ellipsis, PencilLine, Plus, Trash2 } from '@lucide/vue'
import { ref } from 'vue'

const props = defineProps<{
  tabs: Category[]
  activeGuid: string
}>()

const emit = defineEmits<{
  add: []
  select: [guid: string]
  rename: [guid: string, name: string]
  remove: [guid: string, name: string]
  reorder: [from: number, to: number]
}>()

// Drag: highlight the source (opacity) and the insertion target (ring) while
// dragging — no live DOM reordering, so no flicker. The real reorder happens
// once on drop (animated via TransitionGroup move).
const dragIndex = ref<number | null>(null)
const fromIndex = ref<number | null>(null)
const overIndex = ref<number | null>(null)

function onDragStart(index: number) {
  fromIndex.value = index
  dragIndex.value = index
  overIndex.value = null
}

function onDragOver(e: Event, index: number) {
  e.preventDefault()
  if (dragIndex.value === null)
    return
  overIndex.value = index
}

function onDrop(e: Event) {
  e.preventDefault()
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
</script>

<template>
  <div
    class="flex items-center gap-1 border-b border-gray-300 bg-white px-2 py-1.5 dark:border-gray-700 dark:bg-gray-800"
  >
    <TransitionGroup name="list" class="flex items-center gap-1">
      <div
        v-for="(tab, index) in tabs" :key="tab.guid" :draggable="true"
        class="group flex cursor-pointer items-center gap-1 rounded px-2.5 py-1 select-none" :class="[
          tab.guid === activeGuid
            ? 'bg-blue-500 text-white'
            : 'text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-700',
          dragIndex === index ? 'opacity-40' : '',
          overIndex === index && dragIndex !== null && overIndex !== dragIndex ? 'ring-2 ring-blue-400' : '',
        ]" @click="emit('select', tab.guid)" @dragstart="onDragStart(index)" @dragover="onDragOver($event, index)"
        @drop="onDrop($event)" @dragend="onDragEnd"
      >
        <span class="text-sm">{{ tab.name }}</span>

        <Menu as="div" class="relative">
          <MenuButton
            class="flex h-4 w-4 items-center justify-center rounded text-xs opacity-0 hover:bg-white/20 group-hover:opacity-100"
            @click.stop
          >
            <Ellipsis class="h-3 w-3" />
          </MenuButton>
          <TransitionRoot
            enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
            enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in" leave-from="opacity-100 scale-100"
            leave-to="opacity-0 scale-95"
          >
            <MenuItems
              class="absolute left-0 z-10 mt-1 w-32 origin-top-left rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
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
          </TransitionRoot>
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
