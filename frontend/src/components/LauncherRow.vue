<script setup lang="ts">
import type { AppItem } from '../api'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { Copy, Ellipsis, Folder, Pencil, PencilLine, Trash2 } from '@lucide/vue'
import { computed } from 'vue'
import { ConvertItemToAbsolute, ConvertItemToRelative, Launch, Reveal, Stop, UpdateIcon } from '../api'
import { useMenuFlip } from '../composables/useMenuFlip'
import { formatRuntime, isAutoIcon, showError } from '../utils'

const props = defineProps<{
  item: AppItem
  iconUrl?: string
  running?: boolean
  runtimeMs?: number
  gameMode?: boolean
  dragging?: boolean
  dragOver?: boolean
}>()

const emit = defineEmits<{
  'rename': []
  'details': []
  'duplicate': []
  'delete': []
  'refresh': []
  'icondone': [icon: string, iconUrl: string]
  'edit-runtime': []
  'dragstart': [e: DragEvent]
  'dragover': []
  'drop': []
  'dragend': []
}>()

const runtimeText = computed(() => formatRuntime(props.runtimeMs ?? 0))

// Flip the row menu upward when there is not enough room below, so it never
// overflows the window and causes a global scrollbar.
const { onMenuButtonClick, menuPosition } = useMenuFlip({ estimate: 260 })

async function onRun() {
  try {
    if (props.running)
      await Stop(props.item.guid)
    else
      await Launch(props.item.guid)
  }
  catch (err) {
    showError(err)
  }
}

function onDoubleClick(e: MouseEvent) {
  if (props.running)
    return
  const target = e.target as HTMLElement | null
  // Only launch when double-clicking non-interactive areas (not buttons / runtime edit).
  if (target?.closest('button'))
    return
  if (target?.closest('[data-runtime-edit]'))
    return
  Launch(props.item.guid).catch(showError)
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
    class="mb-1.5 flex select-none items-center gap-2.5 rounded border border-gray-300 bg-white px-2.5 py-1.5 dark:border-gray-700 dark:bg-gray-800"
    :class="{
      'opacity-40': dragging,
      'ring-2 ring-blue-400': dragOver,
    }" @dblclick="onDoubleClick" @dragstart="emit('dragstart', $event)" @dragover.prevent="emit('dragover')"
    @drop.prevent="emit('drop')" @dragend="emit('dragend')"
  >
    <img
      v-if="iconUrl || item.icon || undefined" :src="iconUrl || item.icon || undefined" alt=""
      class="h-7 w-7 shrink-0 object-contain"
    >
    <span v-else class="h-7 w-7 shrink-0 object-contain" />
    <span class="min-w-0 flex-1 truncate">{{ item.name }}</span>
    <span
      v-if="gameMode" data-runtime-edit class="shrink-0 rounded px-0.5 text-gray-500 dark:text-gray-400"
      :class="running ? 'cursor-not-allowed' : 'cursor-pointer hover:text-blue-600 hover:underline dark:hover:text-blue-400'"
      :title="running ? 'Running — stop the program to edit runtime' : 'Click to edit runtime'"
      @click="!running && emit('edit-runtime')"
    >{{ runtimeText }}</span>
    <button
      class="rounded px-2.5 py-1 cursor-pointer"
      :class="running ? 'bg-red-500 text-white hover:bg-red-600' : 'text-gray-700 hover:bg-gray-200 dark:text-gray-100 dark:hover:bg-gray-700'"
      @click="onRun"
    >
      {{ running ? 'Stop' : 'Run' }}
    </button>

    <Menu as="div" class="relative">
      <MenuButton
        class="cursor-pointer rounded p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
        @click="onMenuButtonClick"
      >
        <Ellipsis class="h-4 w-4" />
      </MenuButton>
      <Teleport to="body">
        <MenuItems
          class="w-54 overflow-y-auto rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          :style="menuPosition('right')"
        >
          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="emit('details')"
            >
              <Pencil class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
              <span>Edit</span>
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="emit('duplicate')"
            >
              <Copy class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
              <span>Duplicate</span>
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="run(() => Reveal(props.item.guid))"
            >
              <Folder class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
              <span>Open folder...</span>
            </button>
          </MenuItem>
          <div class="my-1 border-t border-gray-200 dark:border-gray-700" />

          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="emit('rename')"
            >
              <PencilLine class="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
              <span>Rename</span>
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-red-600 dark:text-red-400"
              :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''" @click="emit('delete')"
            >
              <Trash2 class="h-4 w-4 shrink-0" />
              <span>Delete</span>
            </button>
          </MenuItem>
          <div class="my-1 border-t border-gray-200 dark:border-gray-700" />

          <MenuItem v-if="isAutoIcon(props.item.icon)" v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="run(async () => { const res = await UpdateIcon(props.item.guid); emit('icondone', res.icon, res.icon_url) })"
            >
              <span>Update icon</span>
            </button>
          </MenuItem>

          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="run(async () => { await ConvertItemToAbsolute(props.item.guid); emit('refresh') })"
            >
              <span>Convert to absolute path</span>
            </button>
          </MenuItem>
          <MenuItem v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left" :class="active ? 'bg-gray-100 dark:bg-gray-700' : ''"
              @click="run(async () => { await ConvertItemToRelative(props.item.guid); emit('refresh') })"
            >
              <span>Convert to relative path</span>
            </button>
          </MenuItem>
        </MenuItems>
      </Teleport>
    </Menu>
  </div>
</template>
