import { send } from './client.ts'
import { readList, readNumber, readString } from './read.ts'

export type Category = { id: string; name: string; parentId: string }

export type Brand = { id: string; name: string }

export type Recommendation = { id: string; productId: string; name: string; image: string; minPrice: number }

export type Review = { id: string; stars: number; content: string; displayName: string; specs: string }

export type StockHint = { skuId: string; available: number }

export function listCategories() {
  return send('/v1/categories', {})
}

export function listRecommendations(slot: string) {
  return send(`/v1/recommendations?slot=${encodeURIComponent(slot)}`, {})
}

export function listReviews(productId: string) {
  return send(`/v1/products/${encodeURIComponent(productId)}/reviews`, {})
}

export function createReview(input: { orderItemId: string; productId: string; stars: number; content: string; anonymous: boolean }) {
  return send('/v1/reviews', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      order_item_id: input.orderItemId,
      product_id: input.productId,
      stars: input.stars,
      content: input.content,
      anonymous: input.anonymous,
    }),
  })
}

export function getStocks(skuIds: string[]) {
  const query = skuIds.map((id) => `sku_ids=${encodeURIComponent(id)}`).join('&')
  return send(`/v1/stocks?${query}`, {})
}

export function listBrands() {
  return send('/v1/brands', {})
}

export function readBrands(body: unknown): Brand[] {
  return readList(body, 'brands').flatMap((row) => {
    const id = readString(row, 'id')
    if (!id) return []
    return [{ id, name: readString(row, 'name') }]
  })
}

export function readCategories(body: unknown): Category[] {
  const out: Category[] = []
  for (const row of readList(body, 'categories')) {
    walk(row, '', out)
  }
  return out
}

// 嵌套子分类没带 parent_id 时用外层分类补上
function walk(row: Record<string, unknown>, parentId: string, out: Category[]) {
  const id = readString(row, 'id')
  if (id) out.push({ id, name: readString(row, 'name'), parentId: readString(row, 'parent_id', 'parentId') || parentId })
  if (Array.isArray(row.children)) {
    for (const child of row.children) {
      if (child && typeof child === 'object' && !Array.isArray(child)) walk(child as Record<string, unknown>, id, out)
    }
  }
}

// 按 parentId 收成两级树 找不到父级的也当一级
export function categoryTree(list: Category[]): { root: Category; children: Category[] }[] {
  const seen = new Map<string, Category>()
  for (const item of list) if (!seen.has(item.id)) seen.set(item.id, item)
  const all = [...seen.values()]
  const roots = all.filter((item) => !item.parentId || !seen.has(item.parentId))
  return roots.map((root) => ({ root, children: all.filter((item) => item.parentId === root.id) }))
}

export function readRecommendations(body: unknown): Recommendation[] {
  return readList(body, 'recommendations').flatMap((row) => {
    const product = row.product && typeof row.product === 'object' ? (row.product as Record<string, unknown>) : row
    const productId = readString(row, 'product_id', 'productId') || readString(product, 'id')
    if (!productId) return []
    return [{
      id: readString(row, 'id') || productId,
      productId,
      name: readString(product, 'name'),
      image: readString(product, 'main_image', 'mainImage'),
      minPrice: readNumber(product, 'min_price', 'minPrice'),
    }]
  })
}

export function readReviews(body: unknown): Review[] {
  return readList(body, 'reviews').flatMap((row) => {
    const id = readString(row, 'id')
    if (!id) return []
    return [{
      id,
      stars: readNumber(row, 'stars'),
      content: readString(row, 'content'),
      displayName: readString(row, 'display_name', 'displayName') || '用户',
      specs: readString(row, 'specs_json', 'specsJson'),
    }]
  })
}

export function readStocks(body: unknown): StockHint[] {
  return readList(body, 'stocks').flatMap((row) => {
    const skuId = readString(row, 'sku_id', 'skuId')
    if (!skuId) return []
    return [{ skuId, available: readNumber(row, 'available') }]
  })
}
