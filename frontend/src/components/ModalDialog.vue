<script setup lang="ts">
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  TransitionChild,
  TransitionRoot,
} from '@headlessui/vue'
import { ref, watch } from 'vue'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'
import { Details, RenameItem } from '../api'
import { showError } from '../utils'

const props = defineProps<{
  open: boolean
  index: number
  mode: 'rename' | 'details'
  title: string
  initialName: string
}>()

const emit = defineEmits<{
  close: []
}>()

const name = ref('')
const details = ref('')

watch(
  () => [props.open, props.mode, props.index, props.initialName] as const,
  async ([open, mode, , initialName]) => {
    if (!open)
      return
    name.value = mode === 'rename' ? initialName : ''
    details.value = ''
    if (mode === 'details') {
      try {
        details.value = await Details(props.index)
      }
      catch (err) {
        showError(err)
      }
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
  <TransitionRoot
    :show="open"
    as="template"
    enter="duration-200 ease-out"
    enter-from="opacity-0"
    enter-to="opacity-100"
    leave="duration-150 ease-in"
    leave-from="opacity-100"
    leave-to="opacity-0"
  >
    <Dialog
      :open="open"
      class="relative z-20"
      @close="emit('close')"
    >
      <TransitionChild
        as="template"
        enter="duration-200 ease-out"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="duration-150 ease-in"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div
          class="fixed inset-0 bg-black/40"
          aria-hidden="true"
        />
      </TransitionChild>

      <TransitionChild
        as="template"
        enter="duration-200 ease-out"
        enter-from="opacity-0 scale-95"
        enter-to="opacity-100 scale-100"
        leave="duration-150 ease-in"
        leave-from="opacity-100 scale-100"
        leave-to="opacity-0 scale-95"
      >
        <div class="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel class="w-[420px] max-w-full rounded-md bg-white p-4 dark:bg-gray-800 dark:text-gray-100">
            <DialogTitle
              as="h3"
              class="mb-2.5 mt-0"
            >
              {{ title }}
            </DialogTitle>
            <input
              v-if="mode === 'rename'"
              v-model="name"
              type="text"
              autofocus
              class="w-full rounded border border-gray-400 px-1.5 py-1 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
              @keyup.enter="onOk"
            >
            <template v-else>
              <textarea
                v-model="details"
                readonly
                class="mb-2 h-44 w-full resize-none font-mono text-xs dark:bg-gray-700 dark:text-gray-100"
              />
              <button
                class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
                @click="onCopy"
              >
                Copy to Clipboard
              </button>
            </template>
            <div class="mt-3 flex justify-end gap-2">
              <button
                v-if="mode === 'rename'"
                class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
                @click="onOk"
              >
                OK
              </button>
              <button
                class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 dark:hover:bg-gray-600"
                @click="emit('close')"
              >
                Cancel
              </button>
            </div>
          </DialogPanel>
        </div>
      </TransitionChild>
    </Dialog>
  </TransitionRoot>
</template>
