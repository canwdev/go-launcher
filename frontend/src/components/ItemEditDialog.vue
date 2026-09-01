<script setup lang="ts">
import type { AppItem } from '../api'
import { ref, watch } from 'vue'
import { PickDirectory, PickFile, PickImageFile } from '../api'
import AppDialog from './AppDialog.vue'

const props = defineProps<{
  open: boolean
  item: AppItem | null
}>()

const emit = defineEmits<{
  close: []
  save: [guid: string, fields: { name: string, path: string, args: string, working_dir: string, icon: string }]
}>()

const name = ref('')
const path = ref('')
const args = ref('')
const workingDir = ref('')
const icon = ref('')

watch(
  () => [props.open, props.item] as const,
  () => {
    if (!props.open || !props.item)
      return
    name.value = props.item.name ?? ''
    path.value = props.item.path ?? ''
    args.value = props.item.args ?? ''
    workingDir.value = props.item.working_dir ?? ''
    icon.value = props.item.icon ?? ''
  },
)

function dirname(p: string): string {
  const norm = p.replace(/[\\/]+$/, '')
  const idx = Math.max(norm.lastIndexOf('\\'), norm.lastIndexOf('/'))
  return idx >= 0 ? norm.slice(0, idx) : p
}

async function browsePath() {
  const sel = await PickFile(dirname(path.value))
  if (sel)
    path.value = sel
}

async function browseDir() {
  const sel = await PickDirectory(workingDir.value || dirname(path.value))
  if (sel)
    workingDir.value = sel
}

async function browseIcon() {
  const sel = await PickImageFile(icon.value ? dirname(icon.value) : dirname(path.value))
  if (sel)
    icon.value = sel
}

function onSave() {
  if (!props.item)
    return
  emit('save', props.item.guid, {
    name: name.value.trim(),
    path: path.value.trim(),
    args: args.value.trim(),
    working_dir: workingDir.value.trim(),
    icon: icon.value.trim(),
  })
  emit('close')
}
</script>

<template>
  <AppDialog :open="open" title="Edit item" @close="emit('close')">
    <form id="item-edit-form" class="flex flex-col gap-3" @submit.prevent="onSave">
      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">ID</span>
        <input
          :value="props.item?.guid ?? ''" type="text" readonly
          class="w-full rounded border border-gray-400 bg-gray-100 px-1.5 py-1 text-gray-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-400"
        >
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Name</span>
        <input
          v-model="name" type="text"
          class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        >
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Path</span>
        <div class="flex gap-1">
          <input
            v-model="path" type="text"
            class="flex-1 rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
          >
          <button
            type="button"
            class="shrink-0 rounded border border-gray-400 bg-white px-2 py-1 text-xs hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
            @click="browsePath"
          >
            Browse…
          </button>
        </div>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Args</span>
        <input
          v-model="args" type="text" placeholder="optional"
          class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        >
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Working directory</span>
        <div class="flex gap-1">
          <input
            v-model="workingDir" type="text"
            class="flex-1 rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
          >
          <button
            type="button"
            class="shrink-0 rounded border border-gray-400 bg-white px-2 py-1 text-xs hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
            @click="browseDir"
          >
            Browse…
          </button>
        </div>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs text-gray-500 dark:text-gray-400">Icon</span>
        <div class="flex gap-1">
          <input
            v-model="icon" type="text" placeholder="auto-generated"
            class="flex-1 rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
          >
          <button
            type="button"
            class="shrink-0 rounded border border-gray-400 bg-white px-2 py-1 text-xs hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
            @click="browseIcon"
          >
            Browse…
          </button>
          <button
            type="button"
            class="shrink-0 rounded border border-gray-400 bg-white px-2 py-1 text-xs hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
            @click="icon = ''"
          >
            Clear
          </button>
        </div>
      </label>
    </form>

    <template #actions>
      <button
        type="submit" form="item-edit-form"
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
