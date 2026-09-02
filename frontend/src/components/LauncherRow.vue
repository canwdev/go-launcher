<script setup lang="ts">
import type { AppItem } from '../api'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { Clock, Copy, Ellipsis, Folder, Pencil, PencilLine, Plus, Timer, Trash2 } from '@lucide/vue'
import { computed } from 'vue'
import { ConvertItemToAbsolute, ConvertItemToRelative, Launch, Open, Reveal, Stop, UpdateIcon } from '../api'
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
  timerActive?: boolean
  /** 计时分钟展示串（如 +1m / +... / --），由 App 层格式化 */
  timerMinutes?: string
  /** 该 item 是否启用 autoTimer（启动后自动触发手动计时，不跟踪进程） */
  autoTimer?: boolean
}>()

const emit = defineEmits<{
  'rename': []
  'details': []
  'duplicate': []
  'delete': []
  'refresh': []
  'icondone': [icon: string, iconUrl: string]
  'edit-runtime': []
  'stop-timer': []
  'launched': []
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
    if (props.running) {
      // 计时中允许 Stop（否则 auto_timer 启动后程序将无法停止）
      await Stop(props.item.guid)
    }
    else {
      // autoTimer 项：不跟踪进程（无 Stop / 无自动最小化），仅启动
      if (props.autoTimer)
        await Open(props.item.guid)
      else
        await Launch(props.item.guid)
      emit('launched')
    }
  }
  catch (err) {
    showError(err)
  }
}

function onDoubleClick(e: MouseEvent) {
  // Manual timer running: no launching at all (button and dbl-click both blocked).
  if (props.timerActive || props.running)
    return
  const target = e.target as HTMLElement | null
  // Only launch when double-clicking non-interactive areas (not buttons / runtime edit).
  if (target?.closest('button'))
    return
  if (target?.closest('[data-runtime-edit]'))
    return
  const p = props.autoTimer ? Open(props.item.guid) : Launch(props.item.guid)
  p.then(() => emit('launched')).catch(showError)
}

async function run(action: () => Promise<void>) {
  try {
    await action()
  }
  catch (err) {
    showError(err)
  }
}

// Runtime text click: while a manual timer is active the click stops the timer
// (and saves) — even if the program is running, otherwise auto_timer would deadlock
// the row. Otherwise running disables editing; else it opens the edit dialog.
function onRuntimeClick() {
  if (props.timerActive) {
    emit('stop-timer')
    return
  }
  if (props.running)
    return
  emit('edit-runtime')
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
      v-if="gameMode" data-runtime-edit class="inline-flex shrink-0 items-center gap-0.5 rounded px-0.5"
      :class="timerActive
        ? 'text-red-500 hover:text-red-600'
        : (running ? 'cursor-not-allowed text-gray-500 dark:text-gray-400' : 'text-gray-500 hover:text-blue-600 hover:underline dark:text-gray-400 dark:hover:text-blue-400')"
      :title="timerActive ? 'Click to stop timer and save' : (running ? 'Running — stop the program to edit runtime' : 'Click to edit runtime')"
      @click="onRuntimeClick"
    ><Timer v-if="autoTimer" class="h-3 w-3 shrink-0" /><Clock v-else class="h-3 w-3 shrink-0" />{{ runtimeText }}<template v-if="timerActive"> (<Plus class="h-3 w-3" />{{ timerMinutes }})</template></span>
    <button
      class="rounded px-2.5 py-1"
      :class="(running ? 'bg-red-500 text-white hover:bg-red-600' : 'text-gray-700 hover:bg-gray-200 dark:text-gray-100 dark:hover:bg-gray-700')"
      @click="onRun"
    >
      {{ running ? 'Stop' : 'Run' }}
    </button>

    <Menu as="div" class="relative">
      <MenuButton
        class="rounded p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
        :class="timerActive ? 'cursor-not-allowed opacity-50 disabled:hover:bg-transparent dark:disabled:hover:bg-transparent' : ''"
        :disabled="timerActive"
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
