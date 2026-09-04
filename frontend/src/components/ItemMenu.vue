<script setup lang="ts">
import type { MenuEntry } from '../composables/itemMenu'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { ref } from 'vue'
import { useMenuFlip } from '../composables/useMenuFlip'

const props = withDefaults(defineProps<{
  entries: MenuEntry[]
  disabled?: boolean
  estimate?: number
  align?: 'left' | 'right'
  widthClass?: string
}>(), {
  disabled: false,
  estimate: 260,
  align: 'right',
  widthClass: 'w-54',
})

const { onMenuButtonClick, onMenuOpenAt, menuPosition } = useMenuFlip({ estimate: props.estimate })

const btnWrapRef = ref<HTMLElement | null>(null)
// 右键程序化打开时跳过按钮坐标覆盖（锚点已由鼠标位置设定）
let skipButtonAnchor = false

function onBtnClick(e: MouseEvent) {
  if (skipButtonAnchor) {
    skipButtonAnchor = false
    return
  }
  onMenuButtonClick(e)
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
  <Menu v-slot="{ open }" as="div" class="relative">
    <div ref="btnWrapRef" data-item-menu-btn class="inline-block">
      <MenuButton
        class="rounded p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
        :class="disabled ? 'cursor-not-allowed opacity-50 disabled:hover:bg-transparent dark:disabled:hover:bg-transparent' : ''"
        :disabled="disabled"
        @click.stop="onBtnClick"
      >
        <slot name="button" />
      </MenuButton>
    </div>
    <Teleport to="body">
      <!-- 全屏透明遮罩：菜单打开期间，点击任意处仅关闭菜单，不触发 item 的点击执行。
           遮罩是 body 直接子元素，点击事件不会冒泡到 item 卡片。 -->
      <div v-if="open" class="fixed inset-0 z-40" @contextmenu.prevent />
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
            </button>
          </MenuItem>
        </template>
      </MenuItems>
    </Teleport>
  </Menu>
</template>
