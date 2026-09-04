import type { AppItem } from '../api'
import type { useManualTimer } from './useManualTimer'
import type { useStore } from './useStore'
import { ref } from 'vue'
import { showError } from '../utils'
import { useConfirmDialog } from './useConfirmDialog'
import { useModalDialog } from './useModalDialog'
import { showToast } from './useToast'

type StoreApi = ReturnType<typeof useStore>

export interface ItemEditFields {
  name: string
  path: string
  args: string
  working_dir: string
  icon: string
}

/**
 * 集中管理应用内所有模态交互：重命名弹窗、确认弹窗、Item 编辑/新建、
 * Runtime 编辑、手动计时触发（start/stop/auto）。让 App.vue 不再承担
 * 对话框状态与回调的维护。
 *
 * 依赖 useStore 的返回值（同一实例）与 useManualTimer，由 App.vue 注入。
 */
export function useDialogs(storeApi: StoreApi, timer: ReturnType<typeof useManualTimer>) {
  const { store, state, renameItem, renameTab, removeItem, removeTab, updateItem, createItem, setRuntimeMs } = storeApi

  type ModalTarget = { kind: 'item', guid: string } | { kind: 'tab', guid: string }
  const modalTarget = ref<ModalTarget | null>(null)

  function onSubmitModal(name: string) {
    const target = modalTarget.value
    if (!target)
      return
    if (target.kind === 'item')
      renameItem(target.guid, name).catch(showError)
    else
      renameTab(target.guid, name).catch(showError)
  }

  const { open: modalOpen, title: modalTitle, name: modalName, openRename: openModalRename, ok: onModalOk, close: closeModal } = useModalDialog(onSubmitModal)
  const { open: confirmOpen, message: confirmMessage, request: requestConfirm, requestAsync, confirm: onConfirm, close: closeConfirm } = useConfirmDialog()

  const editOpen = ref(false)
  const editCreating = ref(false)
  const editingItem = ref<AppItem | null>(null)

  function openItemEdit(item: AppItem) {
    editingItem.value = item
    editCreating.value = false
    editOpen.value = true
  }

  function openCreateItem() {
    editingItem.value = null
    editCreating.value = true
    editOpen.value = true
  }

  function onItemSaved(guid: string, fields: ItemEditFields) {
    if (guid)
      updateItem(guid, fields).catch(showError)
    else
      createItem(fields).catch(showError)
  }

  const runtimeEditOpen = ref(false)
  const runtimeEditGuid = ref<string | null>(null)
  const runtimeEditMs = ref(0)

  function onEditRuntime(guid: string) {
    runtimeEditGuid.value = guid
    runtimeEditMs.value = state.value[guid]?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
    runtimeEditOpen.value = true
  }

  function onRuntimeSaved(minutes: number) {
    if (!runtimeEditGuid.value)
      return
    setRuntimeMs(runtimeEditGuid.value, minutes * 60000).catch(showError)
  }

  // 把计时累计 ms 加到该 item 当前 runtime 后写回后端（SetRuntimeMs 是覆盖写，需前端算总和）。
  async function commitTimerMs(guid: string, ms: number) {
    if (ms <= 0)
      return
    const current = state.value[guid]?.runtime_ms ?? store.value.apps[guid]?.runtime_ms ?? 0
    await setRuntimeMs(guid, current + ms).catch(showError)
  }

  function onStartTimer(guid: string) {
    timer.start(guid)
    runtimeEditOpen.value = false
  }

  function onStopTimer(guid: string) {
    timer.stop(guid, async (ms) => {
      if (ms <= 0) {
        showToast('Timer stopped')
        return
      }
      await commitTimerMs(guid, ms)
      showToast('Runtime saved')
    })
  }

  // 启动成功后，若该 item 开启了 autoTimer，自动触发手动计时（不抢占/不重置其它计时）。
  function onLaunched(guid: string) {
    if (timer.isAutoTimer(guid))
      timer.start(guid)
  }

  function openItemRename(item: AppItem) {
    modalTarget.value = { kind: 'item', guid: item.guid }
    openModalRename('Rename', item.name)
  }

  function openTabRename(guid: string, name: string) {
    modalTarget.value = { kind: 'tab', guid }
    openModalRename('Rename Tab', name)
  }

  function onDeleteRequested(item: AppItem) {
    requestConfirm(`Delete "${item.name}"?`, () => removeItem(item.guid).catch(showError))
  }

  function onDeleteTabRequested(guid: string, name: string) {
    const tab = store.value.categories.find(c => c.guid === guid)
    const hasContent = tab?.slots.some(s => s != null) ?? false
    // 空 tab 直接删除，无需确认
    if (!hasContent) {
      removeTab(guid).catch(showError)
      return
    }
    requestConfirm(`Delete tab "${name}"?`, () => removeTab(guid).catch(showError))
  }

  // grid 视图：停止运行中的程序前需用户确认
  function onConfirmStop(resolve: (ok: boolean) => void) {
    requestAsync('Stop this app?').then(resolve)
  }

  return {
    modalOpen,
    modalTitle,
    modalName,
    openModalRename,
    onModalOk,
    closeModal,
    confirmOpen,
    confirmMessage,
    onConfirm,
    closeConfirm,
    editOpen,
    editCreating,
    editingItem,
    openItemEdit,
    openCreateItem,
    onItemSaved,
    runtimeEditOpen,
    runtimeEditGuid,
    runtimeEditMs,
    onEditRuntime,
    onRuntimeSaved,
    onStartTimer,
    onStopTimer,
    onLaunched,
    openItemRename,
    openTabRename,
    onDeleteRequested,
    onDeleteTabRequested,
    onConfirmStop,
  }
}
