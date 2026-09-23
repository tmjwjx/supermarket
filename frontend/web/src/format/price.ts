// 把分换成带两位小数的人民币
export function formatPrice(cents: number): string {
  const value = Number.isFinite(cents) ? Math.trunc(cents) : 0
  const sign = value < 0 ? '-' : ''
  const abs = Math.abs(value)
  const yuan = Math.floor(abs / 100)
  const fen = String(abs % 100).padStart(2, '0')
  return `${sign}¥${yuan}.${fen}`
}
