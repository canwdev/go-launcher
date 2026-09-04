import type { Component } from 'vue'
import { Copy, Folder, Pencil, PencilLine, SquarePlus, Trash2 } from '@lucide/vue'

export interface MenuEntry {
  key: string
  icon?: Component
  label?: string
  divider?: boolean
  danger?: boolean
  /** 条件显示；false 或 undefined 表示显示（show === false 才隐藏） */
  show?: boolean
  action?: () => void
  /** 勾选态开关项：true 时渲染右侧 Check 图标 */
  toggle?: boolean
  /** 勾选态判断（toggle 项） */
  checked?: () => boolean
}

export interface ItemMenuCallbacks {
  onEdit?: () => void
  onDuplicate?: () => void
  onInsertEmpty?: () => void
  onOpenFolder?: () => void
  onRename?: () => void
  onDelete?: () => void
  onUpdateIcon?: () => void
  onToAbsolute?: () => void
  onToRelative?: () => void
}

/** item 菜单（LauncherRow 与 GridItem 共用）。absolute/relative 项保持无图标（用户要求）。 */
export function buildItemMenu(cb: ItemMenuCallbacks, isAutoIcon = false): MenuEntry[] {
  return [
    { key: 'edit', icon: Pencil, label: 'Edit', action: () => cb.onEdit?.() },
    { key: 'duplicate', icon: Copy, label: 'Duplicate', action: () => cb.onDuplicate?.() },
    { key: 'insert-empty', icon: SquarePlus, label: 'Insert empty', action: () => cb.onInsertEmpty?.() },
    { key: 'open-folder', icon: Folder, label: 'Open folder...', action: () => cb.onOpenFolder?.() },
    { key: 'divider-1', divider: true },
    { key: 'rename', icon: PencilLine, label: 'Rename', action: () => cb.onRename?.() },
    { key: 'delete', icon: Trash2, label: 'Delete', danger: true, action: () => cb.onDelete?.() },
    { key: 'divider-2', divider: true },
    { key: 'update-icon', label: 'Update icon', show: isAutoIcon, action: () => cb.onUpdateIcon?.() },
    { key: 'to-absolute', label: 'Convert to absolute path', action: () => cb.onToAbsolute?.() },
    { key: 'to-relative', label: 'Convert to relative path', action: () => cb.onToRelative?.() },
  ]
}

export interface SlotMenuCallbacks {
  onInsertEmpty?: () => void
  onDeleteEmpty?: () => void
}

/** 空槽菜单（仅 grid 网格中的空槽位） */
export function buildSlotMenu(cb: SlotMenuCallbacks): MenuEntry[] {
  return [
    { key: 'insert-empty', icon: SquarePlus, label: 'Insert empty', action: () => cb.onInsertEmpty?.() },
    { key: 'delete-empty', icon: Trash2, label: 'Delete empty', danger: true, action: () => cb.onDeleteEmpty?.() },
  ]
}
