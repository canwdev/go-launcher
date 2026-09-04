import type { Ref } from 'vue'
import { ref } from 'vue'
import { showError } from '../utils'

export interface GridViewDragDeps {
  /** 被拖拽 item 的 guid（与标签栏共享） */
  dragItemGuid: Ref<string | null>
  /** 当前激活 tab（含完整 slots） */
  getTab: () => { slots: (string | null)[] } | null
  /** 统一重排（含 Ctrl 复制） */
  reorderSlots: (from: number, to: number, copy?: boolean) => Promise<void>
}

/**
 * 网格视图拖拽重排。slot index 即数组索引（空槽也是可见元素），
 * 统一走 useStore.reorderSlots（move / Ctrl 复制），并同步共享的 dragItemGuid。
 */
export function useGridViewDrag(deps: GridViewDragDeps) {
  const gridDrag = ref<{ from: number, isSlot: boolean } | null>(null)
  const gridOver = ref<number | null>(null)
  const gridCopy = ref(false)

  function onGridDragStart(index: number, isSlot: boolean) {
    gridDrag.value = { from: index, isSlot }
    gridOver.value = null
    gridCopy.value = false
    // 同步到标签栏拖拽：记录被拖 item 的 guid（空槽无 guid）
    const slot = deps.getTab()?.slots[index]
    deps.dragItemGuid.value = (!isSlot && slot) ? slot : null
  }

  function onGridDragOver(index: number) {
    if (gridDrag.value)
      gridOver.value = index
  }

  function onGridDrop(index: number, ctrl: boolean) {
    const d = gridDrag.value
    if (d && gridOver.value !== null && d.from !== index) {
      if (ctrl && !d.isSlot) {
        gridCopy.value = true
        deps.reorderSlots(d.from, index, true).catch(showError)
      }
      else {
        deps.reorderSlots(d.from, index, false).catch(showError)
      }
    }
    reset()
  }

  function onGridDragEnd() {
    reset()
  }

  function reset() {
    gridDrag.value = null
    gridOver.value = null
    gridCopy.value = false
    deps.dragItemGuid.value = null
  }

  return { gridDrag, gridOver, gridCopy, onGridDragStart, onGridDragOver, onGridDrop, onGridDragEnd, reset }
}
