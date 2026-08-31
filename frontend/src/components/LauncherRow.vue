<script setup lang="ts">
import type { LauncherItem } from '../api'
import {
  Menu,
  MenuButton,
  MenuItem,
  MenuItems,
  TransitionRoot,
} from '@headlessui/vue'
import { computed } from 'vue'
import { ChangeIcon, Launch, Reveal, Stop, UpdateIcon } from '../api'
import { formatRuntime, showError } from '../utils'

const props = defineProps<{
  item: LauncherItem
  index: number
  dragging?: boolean
  dragOver?: boolean
}>()

const emit = defineEmits<{
  rename: []
  details: []
  delete: []
  dragstart: []
  dragover: []
  drop: []
  dragend: []
}>()

const runtimeText = computed(() => formatRuntime(props.item.runtime_ms))

async function onRun() {
  try {
    if (props.item.running)
      await Stop(props.index)
    else
      await Launch(props.index)
  }
  catch (err) {
    showError(err)
  }
}

function onDoubleClick() {
  if (props.item.running)
    return
  Launch(props.index).catch(showError)
}

async function run(action: () => Promise<void>) {
  try {
    await action()
  }
  catch (err) {
    showError(err)
  }
}
</script>

<template>
  <div
    draggable="true"
    class="mb-1.5 flex select-none items-center gap-2.5 rounded border border-gray-300 bg-white px-2.5 py-1.5"
    :class="{
      'cursor-grab': !dragging,
      'opacity-40': dragging,
      'border-blue-500': dragOver,
    }"
    @dblclick="onDoubleClick"
    @dragstart="emit('dragstart')"
    @dragover.prevent="emit('dragover')"
    @drop.prevent="emit('drop')"
    @dragend="emit('dragend')"
  >
    <img :src="item.iconURL || undefined" alt="" class="h-7 w-7 shrink-0 object-contain">
    <span class="min-w-0 flex-1 truncate" :title="item.title">{{ item.title }}</span>
    <span class="shrink-0 text-gray-500">{{ runtimeText }}</span>
    <button
      class="rounded border px-2.5 py-1"
      :class="item.running ? 'border-red-600 bg-red-500 text-white hover:bg-red-600' : 'border-gray-400 bg-white hover:bg-gray-200'"
      @click="onRun"
    >
      {{ item.running ? 'Stop' : 'Run' }}
    </button>

    <Menu as="div" class="relative">
      <MenuButton class="rounded border border-gray-400 bg-white px-2 py-1 hover:bg-gray-200">
        ⋯
      </MenuButton>
      <TransitionRoot
        enter="transition duration-100 ease-out" enter-from="opacity-0 scale-95"
        enter-to="opacity-100 scale-100" leave="transition duration-75 ease-in" leave-from="opacity-100 scale-100"
        leave-to="opacity-0 scale-95"
      >
        <MenuItems
          class="absolute right-0 z-10 mt-1 w-48 origin-top-right rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none"
        >
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
              @click="run(() => Reveal(props.index))"
            >
              Open containing folder
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
              @click="emit('rename')"
            >
              Rename
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
              @click="run(() => ChangeIcon(props.index))"
            >
              Change icon
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
              @click="run(() => UpdateIcon(props.index))"
            >
              Update icon
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left" :class="active ? 'bg-gray-100' : ''"
              @click="emit('details')"
            >
              Details
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="block w-full px-3 py-1.5 text-left text-red-600" :class="active ? 'bg-gray-100' : ''"
              @click="emit('delete')"
            >
              Delete
            </button>
          </MenuItem>
        </MenuItems>
      </TransitionRoot>
    </Menu>
  </div>
</template>
