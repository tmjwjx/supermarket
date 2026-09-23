import { send } from './client.ts'
import { asRecord, readList, readNumber, readString, unwrap } from './read.ts'

export const statusOnSale = 5
export const statusOff = 6

export type ProductCard = {
  id: string
  name: string
  image: string
  minPrice: number
  marketPrice: number
  sales: number
}

export type Sku = {
  id: string
  specs: string
  price: number
  marketPrice: number
  image: string
  enabled: boolean
}

export type SpecOption = {
  name: string
  values: string[]
}

export type Param = {
  name: string
  value: string
}

export type Product = {
  id: string
  name: string
  subtitle: string
  image: string
  status: number
  detailHtml: string
  skus: Sku[]
  specOptions: SpecOption[]
  params: Param[]
}

export type ProductPage = {
  products: ProductCard[]
  nextPageToken: string
}

export type ListQuery = {
  categoryId?: string
  brandId?: string
  minPrice?: number
  maxPrice?: number
  orderBy?: string
  pageSize?: number
  pageToken?: string
}

export function listProducts(query: ListQuery = {}) {
  const params = new URLSearchParams()
  if (query.categoryId) params.set('category_id', query.categoryId)
  if (query.brandId) params.set('brand_id', query.brandId)
  if (query.minPrice && query.minPrice > 0) params.set('min_price', String(query.minPrice))
  if (query.maxPrice && query.maxPrice > 0) params.set('max_price', String(query.maxPrice))
  if (query.orderBy) params.set('order_by', query.orderBy)
  if (query.pageSize) params.set('page_size', String(query.pageSize))
  if (query.pageToken) params.set('page_token', query.pageToken)
  const text = params.toString()
  return send(`/v1/products${text ? `?${text}` : ''}`, {})
}

export function searchProducts(q: string, pageToken = '', pageSize = 0, orderBy = '') {
  const params = new URLSearchParams({ q })
  if (orderBy) params.set('order_by', orderBy)
  if (pageSize) params.set('page_size', String(pageSize))
  if (pageToken) params.set('page_token', pageToken)
  return send(`/v1/products:search?${params.toString()}`, {})
}

export function getProduct(id: string) {
  return send(`/v1/products/${encodeURIComponent(id)}`, {})
}

export function getProductDetail(id: string) {
  return send(`/v1/products/${encodeURIComponent(id)}/detail`, {})
}

export function readProductDetail(body: unknown): string {
  const record = asRecord(body)
  if (!record) return typeof body === 'string' ? body : ''
  return readString(record, 'detail_html', 'detailHtml')
}

export function readProductCards(body: unknown): ProductCard[] {
  return readList(body, 'products').flatMap((row) => {
    const card = readCard(row)
    return card ? [card] : []
  })
}

export function readProductPage(body: unknown): ProductPage {
  const record = asRecord(body) ?? {}
  return {
    products: readProductCards(body),
    nextPageToken: readString(record, 'next_page_token', 'nextPageToken'),
  }
}

export function readProduct(body: unknown): Product | null {
  const product = unwrap(body, 'product')
  if (!product) return null
  const id = readString(product, 'id')
  if (!id) return null
  const skus = Array.isArray(product.skus)
    ? product.skus.flatMap((item) => {
        const row = asRecord(item)
        if (!row) return []
        const skuId = readString(row, 'id')
        if (!skuId) return []
        const enabled = row.enabled === undefined ? true : row.enabled === true
        return [{
          id: skuId,
          specs: readString(row, 'specs_json', 'specsJson'),
          price: readNumber(row, 'price'),
          marketPrice: readNumber(row, 'market_price', 'marketPrice'),
          image: readString(row, 'image'),
          enabled,
        }]
      })
    : []
  const specOptions = readList(product, 'spec_options').concat(readList(product, 'specOptions')).flatMap((row) => {
    const name = readString(row, 'name')
    const values = Array.isArray(row.values) ? row.values.filter((v): v is string => typeof v === 'string') : []
    return name && values.length ? [{ name, values }] : []
  })
  const params = readList(product, 'params').flatMap((row) => {
    const name = readString(row, 'name')
    return name ? [{ name, value: readString(row, 'value') }] : []
  })
  return {
    id,
    name: readString(product, 'name'),
    subtitle: readString(product, 'subtitle'),
    image: readString(product, 'main_image', 'mainImage'),
    status: readNumber(product, 'status'),
    detailHtml: readString(product, 'detail_html', 'detailHtml'),
    skus,
    specOptions,
    params,
  }
}

function readCard(row: Record<string, unknown>): ProductCard | null {
  const id = readString(row, 'id')
  if (!id) return null
  return {
    id,
    name: readString(row, 'name'),
    image: readString(row, 'main_image', 'mainImage'),
    minPrice: readNumber(row, 'min_price', 'minPrice'),
    marketPrice: readNumber(row, 'market_price', 'marketPrice'),
    sales: readNumber(row, 'sales_count', 'salesCount'),
  }
}
