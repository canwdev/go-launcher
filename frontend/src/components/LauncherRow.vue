<script setup lang="ts">
import type { LauncherItem } from '../api'
import { computed } from 'vue'
import { Launch, Stop } from '../api'
import { formatRuntime, showError } from '../utils'

const props = defineProps<{
  item: LauncherItem
  index: number
}>()

const emit = defineEmits<{
  menu: [position: { x: number, y: number }]
}>()

const runtimeText = computed(() => formatRuntime(props.item.runtime_ms))

async function onRun() {
  try {
    if (props.item.running)
      await Stop(props.index)
    else
      await Launch(props.index)
  }
  catch (err) {
    showError(err)
  }
}

function onDoubleClick() {
  if (props.item.running)
    return
  Launch(props.index).catch(showError)
}

function onMenuButton(event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  emit('menu', { x: rect.left, y: rect.bottom })
}
</script>

<template>
  <div
    class="mb-1.5 flex items-center gap-2.5 rounded border border-gray-300 bg-white px-2.5 py-1.5"
    @dblclick="onDoubleClick"
  >
    <img
      :src="item.iconURL || undefined"
      alt=""
      class="h-7 w-7 shrink-0 object-contain"
    >
    <span
      class="min-w-0 flex-1 truncate"
      :title="item.title"
    >{{ item.title }}</span>
    <span class="shrink-0 text-gray-500">{{ runtimeText }}</span>
    <button
      class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
      :class="item.running ? 'border-red-600 bg-red-500 text-white hover:bg-red-600' : ''"
      @click="onRun"
    >
      {{ item.running ? 'Stop' : 'Run' }}
    </button>
    <button
      class="rounded border border-gray-400 bg-white px-2 py-1 hover:bg-gray-200"
      @click.stop="onMenuButton"
    >
      ⋯
    </button>
  </div>
</template>
