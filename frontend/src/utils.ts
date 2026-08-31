export function formatRuntime(ms: number): string {
  const total = Math.floor(ms / 1000)
  const d = Math.floor(total / 86400)
  const h = Math.floor((total % 86400) / 3600)
  const m = Math.floor((total % 3600) / 60)
  const parts: string[] = []
  if (d > 0)
    parts.push(`${d}d`)
  if (h > 0)
    parts.push(`${h}h`)
  if (m > 0)
    parts.push(`${m}m`)
  if (parts.length === 0)
    parts.push('1m')
  return parts.join(' ')
}

export function showError(err: unknown): void {
  alert(String(err))
}
