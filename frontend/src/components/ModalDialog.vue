<script setup lang="ts">
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  TransitionChild,
  TransitionRoot,
} from '@headlessui/vue'
import { onMounted, ref, watch } from 'vue'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  open: boolean
  mode: 'rename' | 'details'
  title: string
  initialName: string
  detailsText: string
}>()

const emit = defineEmits<{
  ok: [name: string]
  close: []
}>()

const name = ref('')
const details = ref('')

watch(
  () => [props.open, props.mode, props.initialName, props.detailsText] as const,
  () => {
    if (!props.open)
      return
    name.value = props.mode === 'rename' ? props.initialName : ''
    details.value = props.detailsText
  },
)

onMounted(() => {
  name.value = props.mode === 'rename' ? props.initialName : ''
  details.value = props.detailsText
})

function onOk() {
  emit('ok', name.value.trim())
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
        <div class="fixed inset-0 bg-black/40" aria-hidden="true" />
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
            <DialogTitle as="h3" class="mb-2.5 mt-0">
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
