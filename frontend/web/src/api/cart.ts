import { send } from './client.ts'
import { readBool, readList, readNumber, readString } from './read.ts'

export type CartItem = {
  skuId: string
  quantity: number
  checked: boolean
  name: string
  specs: string
  price: number
  image: string
  invalid: boolean
  shortStock: boolean
}

export function listCartItems() {
  return send('/v1/cart/items', {})
}

export function addCartItem(input: { skuId: string; quantity: number }) {
  return send('/v1/cart/items', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sku_id: input.skuId, quantity: input.quantity }),
  })
}

export function updateCartItem(input: { skuId: string; quantity: number; checked: boolean }) {
  return send(`/v1/cart/items/${encodeURIComponent(input.skuId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ quantity: input.quantity, checked: input.checked }),
  })
}

export function removeCartItems(skuIds: string[]) {
  return send('/v1/cart/items:batchDelete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sku_ids: skuIds }),
  })
}

export function readCartItems(body: unknown): CartItem[] {
  return readList(body, 'items').flatMap((row) => {
    const skuId = readString(row, 'sku_id', 'skuId')
    if (!skuId) return []
    return [
      {
        skuId,
        quantity: readNumber(row, 'quantity') || 1,
        checked: readBool(row, 'checked'),
        name: readString(row, 'product_name', 'productName'),
        specs: readString(row, 'specs_json', 'specsJson'),
        price: readNumber(row, 'price'),
        image: readString(row, 'image'),
        invalid: readBool(row, 'invalid'),
        shortStock: readBool(row, 'short_stock', 'shortStock'),
      },
    ]
  })
}
