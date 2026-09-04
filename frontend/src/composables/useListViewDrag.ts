import type { Ref } from 'vue'
import type { AppItem } from '../api'
import { ref } from 'vue'
import { showError } from '../utils'

export const ITEM_DRAG_MIME = 'application/x-go-launcher-item'

export interface ListViewDragDeps {
  /** 被拖拽 item 的 guid（与标签栏共享的拖拽状态） */
  dragItemGuid: Ref<string | null>
  /** 列表视图渲染的行（每行带其对应的真实 slot index） */
  rows: Ref<{ item: AppItem, index: number }[]>
  /** 当前激活 tab（含完整 slots，可能含 null 空槽） */
  getTab: () => { slots: (string | null)[] } | null
  /** 统一重排（含 Ctrl 复制）：from/to 均为 slot index */
  reorderSlots: (from: number, to: number, copy?: boolean) => Promise<void>
}

/**
 * 列表视图拖拽重排。
 *
 * 关键点：rows 只渲染非空 item，v-for 的 index 是「渲染索引」；但 slots 数组
 * 可能含 null 空槽，操作 slots 时必须用真实的 slot index（row.index）。
 * 拖拽状态（draggingSlot / dragFromIndex / dragOverIndex）一律存 slot index，
 * 从模板传入的渲染索引通过 rows[index].index 映射——否则含空槽时重排会错位。
 *
 * onDrop 支持 Ctrl/Meta 复制（与网格视图行为一致）：ctrl=true 时在目标位插入
 * 副本、源保持不变，否则移动源。
 */
export function useListViewDrag(deps: ListViewDragDeps) {
  const draggingSlot = ref<number | null>(null)
  const dragFromIndex = ref<number | null>(null)
  const dragOverIndex = ref<number | null>(null)

  function onDragStart(e: DragEvent, index: number) {
    const row = deps.rows.value[index]
    if (!row)
      return
    deps.dragItemGuid.value = row.item.guid
    dragFromIndex.value = row.index
    draggingSlot.value = row.index
    dragOverIndex.value = null
    e.dataTransfer?.setData(ITEM_DRAG_MIME, row.item.guid)
    if (e.dataTransfer)
      e.dataTransfer.effectAllowed = 'copyMove'
  }

  function onDragOver(index: number) {
    if (draggingSlot.value === null)
      return
    const row = deps.rows.value[index]
    if (!row)
      return
    dragOverIndex.value = row.index
  }

  /** ctrl=true 时复制而非移动（与网格视图一致） */
  function onDrop(ctrl: boolean) {
    const tab = deps.getTab()
    if (tab && dragFromIndex.value !== null && dragOverIndex.value !== null && dragFromIndex.value !== dragOverIndex.value) {
      deps.reorderSlots(dragFromIndex.value, dragOverIndex.value, ctrl).catch(showError)
    }
    reset()
  }

  function onDragEnd() {
    reset()
  }

  function reset() {
    dragFromIndex.value = null
    draggingSlot.value = null
    dragOverIndex.value = null
    deps.dragItemGuid.value = null
  }

  return { draggingSlot, dragFromIndex, dragOverIndex, onDragStart, onDragOver, onDrop, onDragEnd, reset }
}
