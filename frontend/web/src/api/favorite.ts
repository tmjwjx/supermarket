import { send } from './client.ts'
import { readBool, readList, readString } from './read.ts'
import { readProductCards, type ProductCard } from './product.ts'

export type Favorite = {
  productId: string
  product: ProductCard | null
  invalid: boolean
}

export function listFavorites() {
  return send('/v1/favorites', {})
}

export function addFavorite(productId: string) {
  return send('/v1/favorites', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ product_id: productId }),
  })
}

export function removeFavorite(productId: string) {
  return send(`/v1/favorites/${encodeURIComponent(productId)}`, { method: 'DELETE' })
}

export function readFavorites(body: unknown): Favorite[] {
  return readList(body, 'favorites').flatMap((row) => {
    const productId = readString(row, 'product_id', 'productId')
    if (!productId) return []
    const cards = readProductCards({ products: row.product ? [row.product] : [] })
    return [{ productId, product: cards[0] ?? null, invalid: readBool(row, 'invalid') }]
  })
}
