<script setup lang="ts">
import {
  Dialog,
  DialogPanel,
  DialogTitle,
  TransitionChild,
  TransitionRoot,
} from '@headlessui/vue'

defineProps<{
  open: boolean
  title: string
  message: string
}>()

const emit = defineEmits<{
  confirm: []
  close: []
}>()
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
          <DialogPanel class="w-[420px] max-w-full rounded-md bg-white p-4">
            <DialogTitle
              as="h3"
              class="mb-2.5 mt-0"
            >
              {{ title }}
            </DialogTitle>
            <p class="m-0">
              {{ message }}
            </p>
            <div class="mt-3 flex justify-end gap-2">
              <button
                class="rounded border border-gray-400 bg-white px-2.5 py-1 hover:bg-gray-200"
                @click="emit('close')"
              >
                Cancel
              </button>
              <button
                class="rounded border border-red-600 bg-red-500 px-2.5 py-1 text-white hover:bg-red-600"
                @click="emit('confirm')"
              >
                Delete
              </button>
            </div>
          </DialogPanel>
        </div>
      </TransitionChild>
    </Dialog>
  </TransitionRoot>
</template>
