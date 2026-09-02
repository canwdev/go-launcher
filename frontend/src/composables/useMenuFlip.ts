import { ref } from 'vue'

export interface UseMenuFlipOptions {
  /** 预估菜单高度（px）。下方剩余空间小于该值且上方空间更大时向上弹出。 */
  estimate?: number
}

export interface MenuAnchor {
  x: number
  y: number
  w: number
  h: number
}

/**
 * 可复用的菜单弹出逻辑：
 * - 点击按钮时测量其在视口中的位置，下方空间不足且上方足够则向上翻转；
 * - 提供 fixed 定位样式，配合 `<Teleport to="body">` 让菜单脱离滚动容器、
 *   始终展示在窗口最顶层，不被遮挡或裁剪。菜单本身无动画，直接显示/隐藏。
 *
 * 用法（Headless UI Menu）：
 * ```ts
 * const { onMenuButtonClick, menuPosition } = useMenuFlip({ estimate: 260 })
 * ```
 * ```html
 * <MenuButton @click="onMenuButtonClick">…</MenuButton>
 * <Teleport to="body">
 *   <MenuItems class="overflow-y-auto w-48 …" :style="menuPosition('right')">…</MenuItems>
 * </Teleport>
 * ```
 */
export function useMenuFlip(options: UseMenuFlipOptions = {}) {
  const menuUp = ref(false)
  const estimate = options.estimate ?? 260
  const anchor = ref<MenuAnchor | null>(null)

  function onMenuButtonClick(e: MouseEvent) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    anchor.value = { x: rect.left, y: rect.top, w: rect.width, h: rect.height }
    const spaceBelow = window.innerHeight - rect.bottom
    const spaceAbove = rect.top
    menuUp.value = spaceBelow < estimate && spaceAbove > spaceBelow
  }

  /**
   * fixed 定位样式（菜单通过 Teleport 渲染在 body 上）：
   * 底边/顶边贴住按钮，水平方向按 align 对齐按钮左/右边缘。
   * 关键：max-height 动态限制为“按钮到对应视口边缘的可用空间”，
   * 保证菜单始终完整可见、不超出视口，也不会撑出全局滚动条。
   */
  function menuPosition(align: 'left' | 'right' = 'right'): Record<string, string> {
    const a = anchor.value
    if (!a)
      return {}
    const pos: Record<string, string> = {
      position: 'fixed',
      zIndex: '50',
    }
    const gap = 6
    if (menuUp.value) {
      pos.bottom = `${Math.max(0, Math.round(window.innerHeight - a.y))}px`
      pos.maxHeight = `min(calc(${Math.round(a.y)}px - ${gap}px), 95vh)`
    }
    else {
      pos.top = `${Math.round(a.y + a.h + gap)}px`
      pos.maxHeight = `min(calc(100vh - ${Math.round(a.y + a.h + gap)}px), 95vh)`
    }
    if (align === 'right') {
      let right = Math.max(0, Math.round(window.innerWidth - (a.x + a.w)))
      // 水平 clamp：菜单约 220px 宽，其左缘不得超出视口左侧（窄视口/贴右边缘时安全）
      const menuW = 220
      right = Math.min(right, Math.max(0, window.innerWidth - menuW - 4))
      pos.right = `${right}px`
    }
    else {
      pos.left = `${Math.round(a.x)}px`
    }
    return pos
  }

  return { menuUp, onMenuButtonClick, menuPosition }
}
