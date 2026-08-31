<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'
import { Details, RenameItem } from '../api'
import { showError } from '../utils'

const props = defineProps<{
  open: boolean
  index: number
  mode: 'rename' | 'details'
  title: string
}>()

const emit = defineEmits<{
  close: []
}>()

const name = ref('')
const details = ref('')
const input = ref<HTMLInputElement | null>(null)

watch(
  () => [props.open, props.mode, props.index] as const,
  async ([open, mode]) => {
    if (!open)
      return
    name.value = ''
    details.value = ''
    if (mode === 'details') {
      try {
        details.value = await Details(props.index)
      }
      catch (err) {
        showError(err)
      }
    }
    else {
      await nextTick()
      input.value?.focus()
    }
  },
)

async function onOk() {
  if (props.mode === 'rename') {
    try {
      await RenameItem(props.index, name.value)
    }
    catch (err) {
      showError(err)
    }
  }
  emit('close')
}

function onCopy() {
  ClipboardSetText(details.value)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-20 flex items-center justify-center bg-black/40"
      @click.self="emit('close')"
    >
      <div class="w-[420px] max-w-[90vw] rounded-md bg-white p-4">
        <h3 class="mb-2.5 mt-0">
          {{ title }}
        </h3>
        <template v-if="mode === 'rename'">
          <input
            ref="input"
            v-model="name"
            type="text"
            class="w-full rounded border border-gray-400 px-1.5 py-1"
            @keyup.enter="onOk"
          >
        </template>
        <template v-else>
          <textarea
            v-model="details"
            readonly
            class="mb-2 h-44 w-full resize-none font-mono text-xs"
          />
          <button
            class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
            @click="onCopy"
          >
            Copy to Clipboard
          </button>
        </template>
        <div class="mt-3 flex justify-end gap-2">
          <button
            v-if="mode === 'rename'"
            class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
            @click="onOk"
          >
            OK
          </button>
          <button
            class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
            @click="emit('close')"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
