import type { MenuEntry } from './composables/itemMenu'
import { ExternalLink, FilePlus2, FolderOpen, Gamepad2, Images, Pin, RefreshCw } from '@lucide/vue'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

export interface AppMenuCtx {
  getGameMode: () => boolean
  toggleGameMode: () => void
  getAutoHide: () => boolean
  toggleAutoHide: () => void
  getAlwaysOnTop: () => boolean
  toggleAlwaysOnTop: () => void
  getAbsolutePaths: () => boolean
  toggleAbsolutePaths: () => void
  onRefresh: () => void
  onOpenDataDir: () => void
  onConvertAbsolute: () => void
  onConvertRelative: () => void
  onBatchUpdateIcons: () => void
}

/** 右上角全局菜单（从 App.vue 抽出，避免上帝组件里堆菜单配置） */
export function buildAppMenu(ctx: AppMenuCtx): MenuEntry[] {
  return [
    {
      key: 'game-mode',
      toggle: true,
      icon: Gamepad2,
      label: 'Game mode',
      checked: ctx.getGameMode,
      action: ctx.toggleGameMode,
    },
    {
      key: 'auto-hide',
      toggle: true,
      label: 'Auto-hide Launcher',
      checked: ctx.getAutoHide,
      action: ctx.toggleAutoHide,
    },
    {
      key: 'always-on-top',
      toggle: true,
      icon: Pin,
      label: 'Always on top',
      checked: ctx.getAlwaysOnTop,
      action: ctx.toggleAlwaysOnTop,
    },
    {
      key: 'absolute-paths',
      toggle: true,
      label: 'Abs path for new items',
      checked: ctx.getAbsolutePaths,
      action: ctx.toggleAbsolutePaths,
    },
    { key: 'divider-2', divider: true },
    {
      key: 'refresh',
      icon: RefreshCw,
      label: 'Refresh',
      action: ctx.onRefresh,
    },
    {
      key: 'open-dir',
      icon: FolderOpen,
      label: 'Open data directory...',
      action: ctx.onOpenDataDir,
    },
    { key: 'divider-1', divider: true },
    {
      key: 'to-absolute',
      label: 'Convert to absolute path',
      action: ctx.onConvertAbsolute,
    },
    {
      key: 'to-relative',
      label: 'Convert to relative path',
      action: ctx.onConvertRelative,
    },
    {
      key: 'batch-icons',
      icon: Images,
      label: 'Batch update icons',
      action: ctx.onBatchUpdateIcons,
    },
    { key: 'divider-3', divider: true },
    {
      key: 'github',
      icon: ExternalLink,
      label: `v${__APP_VERSION__} | GitHub`,
      action: () => BrowserOpenURL('https://github.com/canwdev/go-launcher'),
    },
  ]
}

export interface AddMenuCtx {
  onAddFiles: () => void
  onCreate: () => void
}

/** TabBar 右侧「+」菜单（添加文件 / 新建） */
export function buildAddMenu(ctx: AddMenuCtx): MenuEntry[] {
  return [
    { key: 'pick', icon: FolderOpen, label: 'Pick files...', action: ctx.onAddFiles },
    { key: 'create', icon: FilePlus2, label: 'Create...', action: ctx.onCreate },
  ]
}
