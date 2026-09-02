<script setup lang="ts">
import type { AppItem } from '../api'
import { Ellipsis, Plus } from '@lucide/vue'
import { computed } from 'vue'
import { ConvertItemToAbsolute, ConvertItemToRelative, Reveal, UpdateIcon } from '../api'
import { buildItemMenu, buildSlotMenu } from '../composables/itemMenu'
import { useItemActions } from '../composables/useItemActions'
import { isAutoIcon, showError } from '../utils'
import ItemMenu from './ItemMenu.vue'

const props = defineProps<{
  item: AppItem | null
  slotIndex: number
  iconUrl?: string
  running?: boolean
  runtimeMs?: number
  gameMode?: boolean
  timerActive?: boolean
  timerMinutes?: string
  autoTimer?: boolean
  dragging?: boolean
  dragOver?: boolean
  dragCopy?: boolean
}>()

const emit = defineEmits<{
  'rename': []
  'details': []
  'duplicate': []
  'insert-empty': []
  'delete-empty': []
  'delete': []
  'refresh': []
  'icondone': [icon: string, iconUrl: string]
  'edit-runtime': []
  'stop-timer': []
  'launched': []
  'grid-dragstart': [index: number, isSlot: boolean]
  'grid-dragover': [index: number]
  'grid-drop': [index: number, ctrl: boolean]
  'grid-dragend': []
}>()

const GRID_SLOT_MIME = 'application/x-gol-grid-slot'
const ITEM_DRAG_MIME = 'application/x-go-launcher-item'

const isSlot = computed(() => props.item == null)

async function run(action: () => Promise<void>) {
  try {
    await action()
  }
  catch (err) {
    showError(err)
  }
}

const { runtimeText, runtimeIcon, onRun, onClick } = useItemActions(
  () => ({
    item: props.item ?? { guid: '' },
    running: props.running ?? false,
    runtimeMs: props.runtimeMs ?? 0,
    gameMode: props.gameMode ?? false,
    timerActive: props.timerActive ?? false,
    timerMinutes: props.timerMinutes ?? '--',
    autoTimer: props.autoTimer ?? false,
  }),
  {
    onLaunched: () => emit('launched'),
    onStopTimer: () => emit('stop-timer'),
    onEditRuntime: () => emit('edit-runtime'),
  },
)

const menuEntries = computed(() => buildItemMenu({
  onEdit: () => emit('details'),
  onDuplicate: () => emit('duplicate'),
  onInsertEmpty: () => emit('insert-empty'),
  onOpenFolder: () => run(() => Reveal(props.item!.guid)),
  onRename: () => emit('rename'),
  onDelete: () => emit('delete'),
  onUpdateIcon: () => run(async () => {
    const res = await UpdateIcon(props.item!.guid)
    emit('icondone', res.icon, res.icon_url)
  }),
  onToAbsolute: () => run(async () => {
    await ConvertItemToAbsolute(props.item!.guid)
    emit('refresh')
  }),
  onToRelative: () => run(async () => {
    await ConvertItemToRelative(props.item!.guid)
    emit('refresh')
  }),
}, props.item ? isAutoIcon(props.item.icon) : false))

const slotMenuEntries = computed(() => buildSlotMenu({
  onInsertEmpty: () => emit('insert-empty'),
  onDeleteEmpty: () => emit('delete-empty'),
}))

function onDragStart(e: DragEvent) {
  if (props.dragging)
    return
  emit('grid-dragstart', props.slotIndex, isSlot.value)
  e.dataTransfer?.setData(GRID_SLOT_MIME, String(props.slotIndex))
  // 同时写入 item MIME，支持拖拽到标签栏移动/复制
  if (!isSlot.value && props.item)
    e.dataTransfer?.setData(ITEM_DRAG_MIME, props.item.guid)
  if (e.dataTransfer)
    e.dataTransfer.effectAllowed = 'copyMove'
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  emit('grid-dragover', props.slotIndex)
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  emit('grid-drop', props.slotIndex, e.ctrlKey)
}

function onDragEnd() {
  emit('grid-dragend')
}
</script>

<template>
  <!-- item 卡片 -->
  <div
    v-if="item" draggable="true"
    class="group relative flex select-none flex-col items-center rounded-lg border border-gray-300 bg-white p-2 pt-4 hover:border-blue-400 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-blue-500"
    :class="{
      'opacity-40': dragging,
      'ring-2 ring-blue-400': dragOver,
    }"
    @click="onRun"
    @dragstart="onDragStart"
    @dragover="onDragOver"
    @drop="onDrop"
    @dragend="onDragEnd"
  >
    <!-- 左上角运行时间 -->
    <span
      v-if="gameMode" data-runtime-edit
      class="absolute left-1.5 top-1.5 inline-flex max-w-[calc(100%-2.5rem)] items-center gap-0.5 rounded px-0.5 text-[11px]"
      :class="timerActive
        ? 'text-red-500 hover:text-red-600'
        : (running ? 'text-gray-500 dark:text-gray-400' : 'text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-500')"
      :title="timerActive ? 'Click to stop timer and save' : (running ? 'Running' : 'Click to edit runtime')"
      @click.stop="onClick"
    ><component :is="runtimeIcon" class="h-3 w-3 shrink-0" /><span class="truncate">{{ runtimeText }}</span><template v-if="timerActive">(<Plus class="h-3 w-3" />{{ timerMinutes }})</template></span>

    <!-- 右上角 hover 菜单 -->
    <div class="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
      <ItemMenu :entries="menuEntries" :disabled="timerActive" :estimate="340">
        <template #button>
          <Ellipsis class="h-4 w-4" />
        </template>
      </ItemMenu>
    </div>

    <!-- 中间图标 64x64 -->
    <img
      v-if="iconUrl || item.icon" :src="iconUrl || item.icon" alt=""
      class="h-16 w-16 shrink-0 object-contain"
    >
    <div v-else class="h-16 w-16 shrink-0 rounded bg-gray-100 dark:bg-gray-700" />

    <!-- 标题 -->
    <span class="mt-1.5 w-full truncate text-center text-xs text-gray-700 dark:text-gray-200">{{ item.name }}</span>
  </div>

  <!-- 空槽 -->
  <div
    v-else
    class="group relative flex h-full min-h-24 select-none flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 text-gray-400 dark:border-gray-600 dark:text-gray-500"
    :class="{ 'ring-2 ring-blue-400': dragOver, 'border-green-400 dark:border-green-500': dragCopy && dragOver }"
    draggable="true"
    @dragstart="onDragStart"
    @dragover="onDragOver"
    @drop="onDrop"
    @dragend="onDragEnd"
  >
    <!-- 右上角 hover 菜单（空槽菜单：插入/删除空白） -->
    <div class="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
      <ItemMenu :entries="slotMenuEntries" :estimate="90">
        <template #button>
          <Ellipsis class="h-4 w-4" />
        </template>
      </ItemMenu>
    </div>
    <Plus class="h-5 w-5" />
  </div>
</template>
