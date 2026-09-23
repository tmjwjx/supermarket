export function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as Record<string, unknown>
}

export function readString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'string' && value !== '') return value
  }
  return ''
}

export function readNumber(record: Record<string, unknown>, ...keys: string[]): number {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value !== '') {
      const n = Number(value)
      if (Number.isFinite(n)) return n
    }
  }
  return 0
}

export function readBool(record: Record<string, unknown>, ...keys: string[]): boolean {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'boolean') return value
  }
  return false
}

export function readList(body: unknown, key: string): Record<string, unknown>[] {
  const record = asRecord(body)
  const list = record?.[key]
  if (!Array.isArray(list)) return []
  return list.flatMap((item) => {
    const row = asRecord(item)
    return row ? [row] : []
  })
}

export function unwrap(body: unknown, key: string): Record<string, unknown> | null {
  const record = asRecord(body)
  if (!record) return null
  return asRecord(record[key]) ?? record
}
