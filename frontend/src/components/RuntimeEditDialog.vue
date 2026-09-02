<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { formatRuntime } from '../utils'
import AppDialog from './AppDialog.vue'

const props = defineProps<{
  open: boolean
  runtimeMs: number
}>()

const emit = defineEmits<{
  close: []
  save: [minutes: number]
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
        Current: {{ currentText }} · input → {{ preview }}
      </p>
    </form>

    <template #actions>
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
    </template>
  </AppDialog>
</template>
