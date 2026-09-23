// 把规格 JSON 摊成 键 值 的短文本 解析不了就原样返回
export function specsText(raw: string, fallback = ''): string {
  if (!raw) return fallback
  try {
    const parsed = JSON.parse(raw) as unknown
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const text = Object.entries(parsed as Record<string, unknown>)
        .map(([key, value]) => `${key} ${String(value)}`)
        .join(' ')
      return text || fallback
    }
  } catch {
    return raw
  }
  return raw
}

// 把规格 JSON 解析成 名称 到 值 的表 解析不了返回空表
export function specsMap(raw: string): Record<string, string> {
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    const out: Record<string, string> = {}
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) out[key] = String(value)
    return out
  } catch {
    return {}
  }
}

// 按库存给出货架提示 available 为 undefined 表示还没查到
export function stockText(available: number | undefined): string {
  if (available === undefined) return ''
  if (available <= 0) return '缺货'
  if (available <= 10) return `仅剩 ${available} 件`
  return '有货'
}
