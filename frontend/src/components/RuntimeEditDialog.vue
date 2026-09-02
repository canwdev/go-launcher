<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Play } from '@lucide/vue'
import { formatRuntime } from '../utils'
import AppDialog from './AppDialog.vue'

const props = defineProps<{
  open: boolean
  runtimeMs: number
  /** 该 item 是否已在手动计时中（用于禁用/提示 Start timer 按钮） */
  timerActive?: boolean
  // autoTimer 特性暂时注释：
  // /** 该 item 启动后是否自动触发手动计时 */
  // autoTimer?: boolean
}>()

const emit = defineEmits<{
  'close': []
  'save': [minutes: number]
  'start-timer': []
  // 'auto-timer': [enabled: boolean] // autoTimer 特性暂时注释
}>()

const minutes = ref('0')

const preview = computed(() => {
  const ms = toMs(minutes.value)
  return ms === 0 ? '0m' : formatRuntime(ms)
})

const currentText = computed(() => formatRuntime(props.runtimeMs ?? 0))

function toMs(value: string): number {
  const n = Math.floor(Number(value) || 0)
  return Math.max(0, n) * 60000
}

watch(
  () => [props.open, props.runtimeMs] as const,
  () => {
    if (!props.open)
      return
    minutes.value = String(Math.max(0, Math.floor((props.runtimeMs ?? 0) / 60000)))
  },
)

function onSave() {
  const m = Math.floor(Number(minutes.value) || 0)
  emit('save', m < 0 ? 0 : m)
  emit('close')
}
</script>

<template>
  <AppDialog :open="open" title="Edit runtime" @close="emit('close')">
    <form id="runtime-edit-form" class="flex flex-col gap-3" @submit.prevent="onSave">
      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Runtime (minutes)</span>
        <input
          v-model="minutes" type="number" min="0" step="60" autofocus
          class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        >
      </label>
      <p class="m-0 text-xs text-gray-500 dark:text-gray-400">
        Current: {{ currentText }} ({{ runtimeMs }}ms) · input → {{ preview }}
      </p>
    </form>

    <template #actions>
      <div class="flex w-full items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="flex items-center gap-1.5 rounded border border-blue-500 bg-white px-2.5 py-1 text-blue-600 hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-400 dark:bg-gray-700 dark:text-blue-300 dark:hover:bg-gray-600"
            :disabled="timerActive" :title="timerActive ? 'Timer already running for this item' : 'Start a manual timer for this item'"
            @click="emit('start-timer')"
          >
            <Play class="h-3.5 w-3.5" />
            Manual timer
          </button>
          <!-- autoTimer 特性暂时注释：
          <div class="flex items-center gap-1.5" :title="autoTimer ? 'Auto-start manual timer when this item launches' : 'Do not auto-start manual timer'">
            <button
              type="button" role="switch" :aria-checked="autoTimer"
              class="relative h-4 w-7 shrink-0 rounded-full transition-colors"
              :class="autoTimer ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'"
              @click="emit('auto-timer', !autoTimer)"
            >
              <span class="absolute left-0.5 top-0.5 h-3 w-3 rounded-full bg-white shadow transition-transform" :class="autoTimer ? 'translate-x-3' : ''" />
            </button>
            <span class="text-xs text-gray-500 dark:text-gray-400">Auto</span>
          </div>
          -->
        </div>
        <div class="flex items-center gap-2">
          <button
            type="submit" form="runtime-edit-form"
            class="rounded border border-blue-500 bg-blue-500 px-2.5 py-1 text-white hover:bg-blue-600"
          >
            Save
          </button>
          <button
            type="button"
            class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
            @click="emit('close')"
          >
            Cancel
          </button>
        </div>
      </div>
    </template>
  </AppDialog>
</template>
