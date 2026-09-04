<script setup lang="ts">
import type { AppItem } from '../api'
import { Ellipsis } from '@lucide/vue'
import { computed, ref } from 'vue'
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
  baselineMs?: number
  gameMode?: boolean
  timerActive?: boolean
  liveMs?: number
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
  'confirm-stop': [resolve: (ok: boolean) => void]
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
    // grid 视图：运行中再次点击需弹窗确认后再停止
    confirmStop: () => new Promise<boolean>(resolve => emit('confirm-stop', resolve)),
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

const itemMenuRef = ref<InstanceType<typeof ItemMenu> | null>(null)

/** 右键弹出 item 菜单（以鼠标位置为锚点）；计时中禁用 */
function onContextMenu(e: MouseEvent) {
  if (props.timerActive)
    return
  itemMenuRef.value?.open(e)
}

const slotMenuRef = ref<InstanceType<typeof ItemMenu> | null>(null)

/** 空槽右键弹出空槽菜单（插入/删除空白） */
function onSlotContextMenu(e: MouseEvent) {
  slotMenuRef.value?.open(e)
}

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
    class="group relative flex cursor-grab aspect-square select-none flex-col items-center justify-center rounded-lg p-2 pt-4 transition-colors duration-150 hover:bg-gray-200/50 dark:hover:bg-gray-800"
    :class="{
      'cursor-grabbing': dragging,
      'opacity-40': dragging,
      'ring-2 ring-blue-400': dragOver,
      'bg-red-50 dark:bg-red-900/15': running,
    }"
    :title="item.name"
    @click="onRun"
    @dragstart="onDragStart"
    @dragover="onDragOver"
    @drop="onDrop" @dragend="onDragEnd"
    @contextmenu.prevent="onContextMenu"
  >
    <!-- 左上角运行时间 -->
    <span
      v-if="gameMode" data-runtime-edit
      class="absolute left-1.5 top-1.5 inline-flex max-w-[calc(100%-2.5rem)] items-center gap-0.5 rounded px-0.5 text-[11px]"
      :class="timerActive
        ? 'text-red-500 hover:text-red-600'
        : (running ? 'text-gray-500 dark:text-gray-400' : 'text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-500')"
      :title="`${runtimeText}\n${timerActive ? 'Click to stop timer and save' : (running ? 'Running' : 'Click to edit runtime')}`"
      @click.stop="onClick"
    ><component :is="runtimeIcon" class="h-3 w-3 shrink-0" /><span class="truncate">{{ runtimeText }}</span></span>

    <!-- 右上角 hover 菜单 -->
    <div class="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
      <ItemMenu ref="itemMenuRef" :entries="menuEntries" :disabled="timerActive" :estimate="340">
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
    <div v-else class="h-16 w-16 shrink-0 rounded " />

    <!-- 标题 -->
    <span class="mt-1.5 w-full truncate text-center text-xs text-gray-700 dark:text-gray-200">{{ item.name }}</span>
  </div>

  <!-- 空槽 -->
  <div
    v-else
    class="group relative flex cursor-grab aspect-square select-none flex-col items-center justify-center rounded-lg  text-gray-400 dark:border-gray-600 dark:text-gray-500"
    :class="{ 'cursor-grabbing': dragging, 'ring-2 ring-blue-400': dragOver, 'border-green-400 dark:border-green-500': dragCopy && dragOver }"
    draggable="true"
    @dragstart="onDragStart"
    @dragover="onDragOver"
    @drop="onDrop"
    @dragend="onDragEnd" @contextmenu.prevent="onSlotContextMenu"
  >
    <!-- 右上角 hover 菜单（空槽菜单：插入/删除空白） -->
    <div class="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
      <ItemMenu ref="slotMenuRef" :entries="slotMenuEntries" :estimate="90">
        <template #button>
          <Ellipsis class="h-4 w-4" />
        </template>
      </ItemMenu>
    </div>
  </div>
</template>
