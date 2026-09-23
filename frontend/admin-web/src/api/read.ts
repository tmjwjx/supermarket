export type Row = Record<string, unknown>

// 网关按 proto 字段名输出下划线键 也兼容驼峰键
function pick(row: Row, key: string): unknown {
  if (key in row) return row[key]
  return row[key.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase())]
}

export function asRow(value: unknown): Row | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as Row
}

export function child(row: Row, key: string): Row {
  return asRow(pick(row, key)) ?? {}
}

export function str(row: Row, key: string): string {
  const value = pick(row, key)
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  return ''
}

// int64 可能被编成字符串
export function num(row: Row, key: string): number {
  const value = pick(row, key)
  if (typeof value === 'number') return value
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

export function flag(row: Row, key: string): boolean {
  return pick(row, key) === true
}

export function rows(row: Row, key: string): Row[] {
  const value = pick(row, key)
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => {
    const next = asRow(item)
    return next ? [next] : []
  })
}

export function strs(row: Row, key: string): string[] {
  const value = pick(row, key)
  if (!Array.isArray(value)) return []
  return value.filter((item): item is string => typeof item === 'string')
}
