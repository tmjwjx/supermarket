import { send } from './client.ts'
import { readNumber, readString, unwrap } from './read.ts'

export type Payment = {
  id: string
  orderId: string
  amount: number
  status: number
}

export function createPayment(orderId: string) {
  return send('/v1/payments', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ order_id: orderId }),
  })
}

export function getPayment(id: string) {
  return send(`/v1/payments/${encodeURIComponent(id)}`, {})
}

export function simulatePayment(id: string, success: boolean) {
  return send(`/v1/payments/${encodeURIComponent(id)}:simulate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ success }),
  })
}

export function readPayment(body: unknown): Payment | null {
  const row = unwrap(body, 'payment')
  if (!row) return null
  const id = readString(row, 'id')
  if (!id) return null
  return {
    id,
    orderId: readString(row, 'order_id', 'orderId'),
    amount: readNumber(row, 'amount'),
    status: readNumber(row, 'status'),
  }
}

export function paymentStatus(status: number): string {
  switch (status) {
    case 1:
      return '待支付'
    case 2:
      return '支付成功'
    case 3:
      return '支付失败'
    case 4:
      return '已关闭'
    case 5:
      return '已退款'
    default:
      return '处理中'
  }
}
