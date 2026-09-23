import { send } from './client.ts'
import { asRecord, readBool, readList, readNumber, readString, unwrap } from './read.ts'

export const orderCompleted = 4

export type OrderLine = {
  id: string
  skuId: string
  productId: string
  name: string
  specs: string
  image: string
  price: number
  quantity: number
  amount: number
  reviewed: boolean
}

export type Order = {
  id: string
  orderNo: string
  status: number
  itemsAmount: number
  payAmount: number
  receiver: string
  phone: string
  address: string
  expiresAt: number
  lines: OrderLine[]
}

export function getOrder(id: string) {
  return send(`/v1/orders/${encodeURIComponent(id)}`, {})
}

export function listOrders() {
  return send('/v1/orders', {})
}

export function createOrder(input: {
  lines: { skuId: string; quantity: number }[]
  addressId: string
  requestId: string
}) {
  return send('/v1/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      lines: input.lines.map((line) => ({ sku_id: line.skuId, quantity: line.quantity })),
      address_id: input.addressId,
      request_id: input.requestId,
    }),
  })
}

export function cancelOrder(id: string) {
  return send(`/v1/orders/${encodeURIComponent(id)}:cancel`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
}

export function confirmOrder(id: string) {
  return send(`/v1/orders/${encodeURIComponent(id)}:confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
}

export function readOrders(body: unknown): Order[] {
  return readList(body, 'orders').flatMap((row) => {
    const order = readOrderRecord(row)
    return order ? [order] : []
  })
}

export function readOrder(body: unknown): Order | null {
  const row = unwrap(body, 'order')
  if (!row) return null
  return readOrderRecord(row)
}

function readOrderRecord(row: Record<string, unknown>): Order | null {
  const id = readString(row, 'id')
  if (!id) return null
  const lines = Array.isArray(row.items)
    ? row.items.flatMap((item) => {
        const line = asRecord(item)
        if (!line) return []
        return [{
          id: readString(line, 'id'),
          skuId: readString(line, 'sku_id', 'skuId'),
          productId: readString(line, 'product_id', 'productId'),
          name: readString(line, 'product_name', 'productName'),
          specs: readString(line, 'specs_json', 'specsJson'),
          image: readString(line, 'image'),
          price: readNumber(line, 'price'),
          quantity: readNumber(line, 'quantity'),
          amount: readNumber(line, 'amount'),
          reviewed: readBool(line, 'reviewed'),
        }]
      })
    : []
  return {
    id,
    orderNo: readString(row, 'order_no', 'orderNo'),
    status: readNumber(row, 'status'),
    itemsAmount: readNumber(row, 'items_amount', 'itemsAmount'),
    payAmount: readNumber(row, 'pay_amount', 'payAmount'),
    receiver: readString(row, 'receiver'),
    phone: readString(row, 'phone'),
    address: readString(row, 'address'),
    expiresAt: readNumber(row, 'expires_at_unix', 'expiresAtUnix'),
    lines,
  }
}

export function orderStatus(status: number): string {
  switch (status) {
    case 1:
      return '待支付'
    case 2:
      return '已支付'
    case 3:
      return '已发货'
    case 4:
      return '已完成'
    case 5:
      return '已取消'
    default:
      return '处理中'
  }
}
