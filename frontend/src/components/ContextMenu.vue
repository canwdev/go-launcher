<script setup lang="ts">
import { ChangeIcon, RemoveItem, Reveal, UpdateIcon } from '../api'
import { showError } from '../utils'

const props = defineProps<{
  index: number
  x: number
  y: number
}>()

const emit = defineEmits<{
  close: []
  rename: []
  details: []
}>()

async function run(action: () => Promise<void>) {
  emit('close')
  try {
    await action()
  }
  catch (err) {
    showError(err)
  }
}
</script>

<template>
  <div
    class="fixed z-10 rounded border border-gray-300 bg-white shadow-md"
    :style="{ left: `${props.x}px`, top: `${props.y}px` }"
    @click.stop
  >
    <button
      class="block w-full px-3 py-1.5 text-left hover:bg-gray-100"
      @click="run(() => Reveal(props.index))"
    >
      Open containing folder
    </button>
    <button
      class="block w-full px-3 py-1.5 text-left hover:bg-gray-100"
      @click="emit('close'); emit('rename')"
    >
      Rename
    </button>
    <button
      class="block w-full px-3 py-1.5 text-left hover:bg-gray-100"
      @click="run(() => ChangeIcon(props.index))"
    >
      Change icon
    </button>
    <button
      class="block w-full px-3 py-1.5 text-left hover:bg-gray-100"
      @click="run(() => UpdateIcon(props.index))"
    >
      Update icon
    </button>
    <button
      class="block w-full px-3 py-1.5 text-left hover:bg-gray-100"
      @click="emit('close'); emit('details')"
    >
      Details
    </button>
    <button
      class="block w-full px-3 py-1.5 text-left text-red-600 hover:bg-gray-100"
      @click="run(() => RemoveItem(props.index))"
    >
      Delete
    </button>
  </div>
</template>
