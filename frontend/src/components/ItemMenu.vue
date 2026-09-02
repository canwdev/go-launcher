<script setup lang="ts">
import type { MenuEntry } from '../composables/itemMenu'
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
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

const { onMenuButtonClick, menuPosition } = useMenuFlip({ estimate: props.estimate })
</script>

<template>
  <Menu as="div" class="relative">
    <MenuButton
      class="rounded p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-700"
      :class="disabled ? 'cursor-not-allowed opacity-50 disabled:hover:bg-transparent dark:disabled:hover:bg-transparent' : ''"
      :disabled="disabled"
      @click.stop="onMenuButtonClick"
    >
      <slot name="button" />
    </MenuButton>
    <Teleport to="body">
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
