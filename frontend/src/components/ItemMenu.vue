<script setup lang="ts">
import type { MenuEntry } from '../composables/itemMenu'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { Check } from '@lucide/vue'
import { ref } from 'vue'
import { useMenuFlip } from '../composables/useMenuFlip'

const props = withDefaults(defineProps<{
  entries: MenuEntry[]
  disabled?: boolean
  estimate?: number
  align?: 'left' | 'right'
  widthClass?: string
  /** 覆盖触发按钮的默认样式 */
  buttonClass?: string
}>(), {
  disabled: false,
  estimate: 260,
  align: 'right',
  widthClass: 'w-54',
  buttonClass: 'rounded p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700',
})

const { onMenuButtonClick, onMenuOpenAt, menuPosition } = useMenuFlip({ estimate: props.estimate })

const btnWrapRef = ref<HTMLElement | null>(null)
// 右键程序化打开时跳过按钮坐标覆盖（锚点已由鼠标位置设定）
let skipButtonAnchor = false
// 右键时跳过随后合成的 click：菜单已开时右键触发器会先 close、此处再 return，
// 避免 click 又把菜单重新打开（菜单开着时右键触发器应关闭菜单）
let suppressNextButtonClick = false

function onBtnClick(e: MouseEvent) {
  if (suppressNextButtonClick) {
    suppressNextButtonClick = false
    return
  }
  if (skipButtonAnchor) {
    skipButtonAnchor = false
    return
  }
  onMenuButtonClick(e)
}

/**
 * 触发器的右键：屏蔽原生上下文菜单。菜单开着时先 close 再跳过随后合成的
 * click（Headless UI 的右键不算外部点击，不处理的话 click 会把它重新打开），
 * 菜单没开时直接返回，交给外层 @contextmenu 调 open(e) 以鼠标为锚点打开。
 */
function onTriggerContextMenu(e: MouseEvent, isOpen: boolean, close: () => void) {
  e.preventDefault()
  if (!isOpen)
    return
  close()
  suppressNextButtonClick = true
}

/** 菜单打开期间点击遮罩：关闭菜单（不触发底层元素的点击） */
function onOverlayClick(close: () => void) {
  close()
}

/** 菜单打开期间在遮罩上右键：同样只关闭菜单，并屏蔽原生上下文菜单 */
function onOverlayContextMenu(e: MouseEvent, close: () => void) {
  e.preventDefault()
  close()
}

/** 以鼠标位置为锚点程序化打开菜单（供 item 右键调用）；不传事件则用按钮定位 */
function open(e?: MouseEvent) {
  if (props.disabled)
    return
  if (e) {
    onMenuOpenAt(e)
    skipButtonAnchor = true
  }
  btnWrapRef.value?.querySelector<HTMLButtonElement>('button')?.click()
}

defineExpose({ open })
</script>

<template>
  <Menu v-slot="{ open: isOpen, close }" as="div" class="relative">
    <div ref="btnWrapRef" data-item-menu-btn class="inline-block">
      <MenuButton
        :class="[
          buttonClass,
          disabled ? 'cursor-not-allowed opacity-50 disabled:hover:bg-transparent dark:disabled:hover:bg-transparent' : '',
        ]"
        :disabled="disabled"
        @click.stop="onBtnClick" @contextmenu="onTriggerContextMenu($event, isOpen, close)"
      >
        <slot name="button" />
      </MenuButton>
    </div>
    <Teleport to="body">
      <!-- 全屏透明遮罩：菜单打开期间，点击任意处仅关闭菜单，不触发 item 的点击执行。
           遮罩是 body 直接子元素，点击事件不会冒泡到 item 卡片。 -->
      <div
        v-if="isOpen" class="fixed inset-0 z-40" @click="onOverlayClick(close)"
        @contextmenu="onOverlayContextMenu($event, close)"
      />
      <MenuItems
        class="overflow-y-auto rounded border border-gray-300 bg-white py-1 shadow-md focus:outline-none dark:border-gray-700 dark:bg-gray-800" :class="[widthClass]"
        :style="menuPosition(align)"
      >
        <template v-for="entry in entries" :key="entry.key">
          <div v-if="entry.divider" class="my-1 border-t border-gray-200 dark:border-gray-700" />
          <MenuItem v-else-if="entry.show !== false" v-slot="{ active }">
            <button
              class="flex w-full items-center gap-2 px-3 py-1.5 text-left"
              :class="[active ? 'bg-gray-100 dark:bg-gray-700' : '', entry.danger ? 'text-red-600 dark:text-red-400' : '']"
              @click="entry.action?.()"
            >
              <span
                v-if="entry.icon" class="inline-flex h-4 w-4 shrink-0 items-center justify-center"
                :class="entry.danger ? 'text-red-600 dark:text-red-400' : 'text-gray-500 dark:text-gray-400'"
              >
                <component :is="entry.icon" class="h-4 w-4" />
              </span>
              <span class="min-w-0 flex-1 truncate">{{ entry.label }}</span>
              <Check v-if="entry.toggle && entry.checked?.()" class="h-4 w-4 shrink-0 text-blue-500" />
            </button>
          </MenuItem>
        </template>
      </MenuItems>
    </Teleport>
  </Menu>
</template>
