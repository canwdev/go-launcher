<script setup lang="ts">
import type { AppItem } from '../api'
import { Ellipsis } from '@lucide/vue'
import { computed, ref } from 'vue'
import { ConvertItemToAbsolute, ConvertItemToRelative, Reveal, UpdateIcon } from '../api'
import { buildItemMenu } from '../composables/itemMenu'
import { useItemActions } from '../composables/useItemActions'
import { isAutoIcon, showError } from '../utils'
import ItemMenu from './ItemMenu.vue'

const props = defineProps<{
  item: AppItem
  iconUrl?: string
  running?: boolean
  baselineMs?: number
  gameMode?: boolean
  dragging?: boolean
  dragOver?: boolean
  timerActive?: boolean
  /** 计时分钟展示串（如 1m / 2h 3m / --），由 App 层格式化 */
  liveMs?: number
  /** 该 item 是否启用 autoTimer（启动后自动触发手动计时，不跟踪进程） */
  autoTimer?: boolean
}>()

const emit = defineEmits<{
  'rename': []
  'details': []
  'duplicate': []
  'insert-empty': []
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

const { runtimeText, runtimeIcon, onRun, onClick } = useItemActions(
  () => ({
    item: props.item,
    running: props.running ?? false,
    baselineMs: props.baselineMs ?? 0,
    gameMode: props.gameMode ?? false,
    timerActive: props.timerActive ?? false,
    liveMs: props.liveMs ?? 0,
    autoTimer: props.autoTimer ?? false,
  }),
  {
    onLaunched: () => emit('launched'),
    onStopTimer: () => emit('stop-timer'),
    onEditRuntime: () => emit('edit-runtime'),
  },
)

async function run(action: () => Promise<void>) {
  try {
    await action()
  }
  catch (err) {
    showError(err)
  }
}

const menuEntries = computed(() => buildItemMenu({
  onEdit: () => emit('details'),
  onDuplicate: () => emit('duplicate'),
  onInsertEmpty: () => emit('insert-empty'),
  onOpenFolder: () => run(() => Reveal(props.item.guid)),
  onRename: () => emit('rename'),
  onDelete: () => emit('delete'),
  onUpdateIcon: () => run(async () => {
    const res = await UpdateIcon(props.item.guid)
    emit('icondone', res.icon, res.icon_url)
  }),
  onToAbsolute: () => run(async () => {
    await ConvertItemToAbsolute(props.item.guid)
    emit('refresh')
  }),
  onToRelative: () => run(async () => {
    await ConvertItemToRelative(props.item.guid)
    emit('refresh')
  }),
}, isAutoIcon(props.item.icon)))

function onDoubleClick(e: MouseEvent) {
  // Process running: no relaunch via double-click.
  if (props.running)
    return
  const target = e.target as HTMLElement | null
  // Only launch when double-clicking non-interactive areas (not buttons / runtime edit).
  if (target?.closest('button'))
    return
  if (target?.closest('[data-runtime-edit]'))
    return
  onRun()
}

const itemMenuRef = ref<InstanceType<typeof ItemMenu> | null>(null)

/** 右键弹出 item 菜单（以鼠标位置为锚点）；计时中禁用 */
function onContextMenu(e: MouseEvent) {
  if (props.timerActive)
    return
  itemMenuRef.value?.open(e)
}
</script>

<template>
  <div
    draggable="true"
    class="flex select-none items-center gap-2.5 rounded px-2.5 py-1.5 transition-colors duration-150 hover:bg-gray-200/50 dark:hover:bg-gray-800"
    :class="{
      'opacity-40': dragging,
      'ring-2 ring-blue-400': dragOver,
    }" @dblclick="onDoubleClick" @dragstart="emit('dragstart', $event)" @dragover.prevent="emit('dragover')"
    @drop.prevent="emit('drop')" @dragend="emit('dragend')" @contextmenu.prevent="onContextMenu"
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
      :title="`${runtimeText}\n${timerActive ? 'Click to stop timer and save' : (running ? 'Running — stop the program to edit runtime' : 'Click to edit runtime')}`"
      @click="onClick"
    ><component :is="runtimeIcon" class="h-3 w-3 shrink-0" />{{ runtimeText }}</span>
    <button
      class="rounded px-2.5 py-1"
      :class="running ? 'bg-red-500 text-white hover:bg-red-600' : 'text-gray-700 hover:bg-gray-200 dark:text-gray-100 dark:hover:bg-gray-700'"
      :title="running ? 'Click to stop' : undefined"
      @click="onRun"
    >
      {{ running ? 'Stop' : 'Run' }}
    </button>

    <ItemMenu ref="itemMenuRef" :entries="menuEntries" :disabled="timerActive" :estimate="260">
      <template #button>
        <Ellipsis class="h-4 w-4" />
      </template>
    </ItemMenu>
  </div>
</template>
