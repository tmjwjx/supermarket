// 金额在接口里一律是分
export function formatCents(cents: number): string {
  const sign = cents < 0 ? '-' : ''
  return `${sign}¥${(Math.abs(cents) / 100).toFixed(2)}`
}

export function centsToYuan(cents: number): string {
  return (cents / 100).toFixed(2)
}

// 按元输入最多两位小数 不合法返回 null
export function yuanToCents(text: string): number | null {
  const value = text.trim()
  if (!/^\d+(\.\d{1,2})?$/.test(value)) return null
  const [whole, frac = ''] = value.split('.')
  return Number(whole) * 100 + Number(frac.padEnd(2, '0'))
}
