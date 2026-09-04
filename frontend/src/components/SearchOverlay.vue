<script setup lang="ts">
import { Search } from '@lucide/vue'
import { useThrottleFn } from '@vueuse/core'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { ItemState } from '../api'
import type { Store } from '../composables/useStore'

const props = defineProps<{
  open: boolean
  store: Store
  state: Record<string, ItemState>
  onLaunch: (guid: string, tabGuid: string) => void
}>()

const emit = defineEmits<{ (e: 'update:open', v: boolean): void }>()

const query = ref('')
const activeIndex = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)
const listEl = ref<HTMLElement | null>(null)
/** 仅键盘导航触发滚动；鼠标 hover 切换高亮时不滚动 */
const keyboardNav = ref(false)

interface SearchHit {
  guid: string
  name: string
  path: string
  iconUrl?: string
  tabGuid: string
  tabName: string
}

/** 全局索引：所有 tab 的所有非空 item（含 tab 归属信息） */
const allItems = computed<SearchHit[]>(() => {
  const hits: SearchHit[] = []
  const s = props.store
  for (const cat of s.categories) {
    for (const slot of cat.slots) {
      if (slot == null)
        continue
      const item = s.apps[slot]
      if (!item)
        continue
      hits.push({
        guid: item.guid,
        name: item.name,
        path: item.path,
        iconUrl: props.state[item.guid]?.icon_url,
        tabGuid: cat.guid,
        tabName: cat.name,
      })
    }
  }
  return hits
})

/** 模糊匹配：name / path 大小写不敏感子串；未输入时不展示列表。排序优先完全匹配 */
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q)
    return []
  const hits = allItems.value.filter(h =>
    h.name.toLowerCase().includes(q) || h.path.toLowerCase().includes(q))
  return hits.sort((a, b) => {
    const rank = (h: SearchHit): number => {
      const n = h.name.toLowerCase()
      const p = h.path.toLowerCase()
      if (n === q)
        return 0 // name 完全匹配
      if (p === q)
        return 1 // path 完全匹配
      if (n.startsWith(q))
        return 2 // name 前缀
      if (p.startsWith(q))
        return 3 // path 前缀
      return 4 // 仅包含
    }
    return rank(a) - rank(b)
  })
})

/** 键盘切换 item 时把高亮项滚动到列表可视区中间（原生平滑滚动 + 节流，避免连按抖动） */
const scrollToActive = useThrottleFn(() => {
  const el = listEl.value
  if (!el)
    return
  const child = el.children[activeIndex.value] as HTMLElement | undefined
  if (!child)
    return
  // 用 boundingRect 差值计算相对列表内容的位置，避免 offsetTop 相对外层 offsetParent 的偏差
  const relTop = child.getBoundingClientRect().top - el.getBoundingClientRect().top + el.scrollTop
  el.scrollTo({
    top: relTop - el.clientHeight / 2 + child.offsetHeight / 2,
    behavior: 'smooth',
  })
}, 50)

watch(query, () => {
  activeIndex.value = 0
})

watch(activeIndex, () => {
  if (keyboardNav.value)
    scrollToActive()
})

watch(() => props.open, async (v) => {
  if (!v)
    return
  query.value = ''
  activeIndex.value = 0
  keyboardNav.value = false
  await nextTick()
  inputEl.value?.focus()
})

/** 组件级 Esc 关闭：浮层打开时，窗口内任意焦点按 Esc 均关闭 */
function onWindowKeydown(e: KeyboardEvent) {
  if (!props.open)
    return
  if (e.key === 'Escape') {
    e.preventDefault()
    emit('update:open', false)
  }
}

onMounted(() => window.addEventListener('keydown', onWindowKeydown))
onUnmounted(() => window.removeEventListener('keydown', onWindowKeydown))

function launch(h: SearchHit) {
  props.onLaunch(h.guid, h.tabGuid)
  emit('update:open', false)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    keyboardNav.value = true
    activeIndex.value = Math.min(activeIndex.value + 1, Math.max(0, filtered.value.length - 1))
  }
  else if (e.key === 'ArrowUp') {
    e.preventDefault()
    keyboardNav.value = true
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
  }
  else if (e.key === 'Enter') {
    const h = filtered.value[activeIndex.value]
    if (h)
      launch(h)
  }
  // Esc 由组件级 window 监听统一处理
}

function onMouseEnter(i: number) {
  keyboardNav.value = false
  activeIndex.value = i
}
</script>

<template>
  <Transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-start justify-center bg-black/40 pt-[12vh]"
      @mousedown.self="emit('update:open', false)"
    >
      <div class="w-[560px] max-w-[92vw] overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800">
        <div class="flex items-center gap-2 border-b border-gray-200 px-3 py-2.5 dark:border-gray-700">
          <Search class="h-4 w-4 shrink-0 text-gray-400" />
          <input
            ref="inputEl"
            v-model="query"
            placeholder="Search programs..."
            class="w-full bg-transparent text-sm outline-none placeholder:text-gray-400"
            @keydown="onKeydown"
          >
          <kbd class="shrink-0 rounded border border-gray-200 px-1.5 py-0.5 text-[11px] text-gray-400 dark:border-gray-600">Esc</kbd>
        </div>

        <div ref="listEl" class="max-h-[50vh] overflow-y-auto p-1">
          <div v-if="!filtered.length" class="px-3 py-6 text-center text-sm text-gray-400">
            {{ query.trim() ? 'No results' : 'Type to search...' }}
          </div>
          <button
            v-for="(h, i) in filtered"
            :key="h.guid"
            type="button"
            class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left transition-colors duration-75"
            :class="i === activeIndex ? 'bg-blue-500 text-white' : 'text-gray-700 hover:bg-gray-200/70 dark:text-gray-200 dark:hover:bg-gray-700'"
            @mousedown.prevent="launch(h)"
            @mouseenter="onMouseEnter(i)"
          >
            <img v-if="h.iconUrl" :src="h.iconUrl" class="h-5 w-5 shrink-0 object-contain" alt="">
            <span v-else class="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-gray-200 text-[10px] font-semibold text-gray-500 dark:bg-gray-700">
              {{ h.name.charAt(0).toUpperCase() }}
            </span>
            <span class="min-w-0 flex-1 truncate text-sm">{{ h.name }}</span>
            <span class="max-w-[40%] truncate text-xs opacity-60">{{ h.path || h.tabName }}</span>
            <span
              class="shrink-0 rounded px-1.5 py-0.5 text-[11px]"
              :class="i === activeIndex ? 'bg-white/20 text-white/90' : 'bg-gray-200/70 text-gray-500 dark:bg-gray-700 dark:text-gray-300'"
            >{{ h.tabName }}</span>
          </button>
        </div>

        <div class="border-t border-gray-200 px-3 py-1.5 text-[11px] text-gray-400 dark:border-gray-700">
          ↑↓ navigate · Enter launch · Esc close
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.08s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
