import { showToast } from './composables/useToast'

export function formatRuntime(ms: number): string {
  const total = Math.floor(ms / 1000)
  const d = Math.floor(total / 86400)
  const h = Math.floor((total % 86400) / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const parts: string[] = []
  if (d > 0)
    parts.push(`${d}d`)
  if (h > 0)
    parts.push(`${h}h`)
  if (m > 0)
    parts.push(`${m}m`)
  if (parts.length === 0) {
    // Sub-minute runtimes: show seconds so small values stay visible
    // (e.g. 2703ms -> "2s") instead of collapsing to 0m / "--".
    if (s > 0)
      parts.push(`1m`)
    else
      parts.push('.')
  }
  return parts.join(' ')
}

export function showError(err: unknown): void {
  showToast(String(err), 'error')
}

export function randomUUID(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

export function debounce<T extends (...args: never[]) => unknown>(fn: T, delay: number): (...args: Parameters<T>) => void {
  let timer: number | undefined
  return (...args) => {
    if (timer)
      window.clearTimeout(timer)
    timer = window.setTimeout(fn as () => void, delay, ...args)
  }
}

export function isAutoIcon(icon: string | undefined): boolean {
  if (!icon)
    return true
  const norm = icon.replace(/\\/g, '/')
  return norm.includes('/go-launcher-data/icons/')
}
