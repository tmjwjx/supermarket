import { send } from './client.ts'
import { readList, readString } from './read.ts'
import { readProductCards, type ProductCard } from './product.ts'

export type BrowseItem = {
  productId: string
  product: ProductCard | null
}

export function listBrowseHistories() {
  return send('/v1/browse-histories', {})
}

export function clearBrowseHistories() {
  return send('/v1/browse-histories', { method: 'DELETE' })
}

export function readBrowseHistories(body: unknown): BrowseItem[] {
  return readList(body, 'histories').flatMap((row) => {
    const productId = readString(row, 'product_id', 'productId')
    if (!productId) return []
    const cards = readProductCards({ products: row.product ? [row.product] : [] })
    return [{ productId, product: cards[0] ?? null }]
  })
}
