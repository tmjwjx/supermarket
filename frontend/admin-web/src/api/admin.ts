import { send } from './client.ts'
import { asRow, child, flag, num, rows, str, strs, type Row } from './read.ts'

export const ProductStatus = {
  Draft: 1,
  Pending: 2,
  Rejected: 3,
  Approved: 4,
  OnSale: 5,
  Off: 6,
} as const

export const productStatusText: Record<number, string> = {
  1: '草稿',
  2: '待审核',
  3: '已驳回',
  4: '已通过',
  5: '在售',
  6: '已下架',
}

export const AttributeKind = { Spec: 1, Param: 2 } as const

export type ProductCard = {
  id: string
  name: string
  mainImage: string
  minPrice: number
  salesCount: number
  status: number
}

export type Sku = {
  id: string
  specs: Record<string, string>
  specsJson: string
  price: number
  marketPrice: number
  image: string
  barcode: string
  enabled: boolean
}

export type ProductParam = { name: string; value: string }

export type Product = {
  id: string
  categoryId: string
  brandId: string
  name: string
  subtitle: string
  keywords: string
  unit: string
  weightGram: number
  status: number
  minPrice: number
  maxPrice: number
  skus: Sku[]
  images: string[]
  detailHtml: string
  params: ProductParam[]
}

export type SkuInput = {
  id?: string
  specs_json: string
  price: number
  market_price: number
  image: string
  barcode: string
  enabled: boolean
}

export type ProductInput = {
  category_id: string
  brand_id: string
  name: string
  subtitle: string
  keywords: string
  unit: string
  weight_gram: number
  images: string[]
  detail_html: string
  skus: SkuInput[]
  params: ProductParam[]
}

export type Page<T> = { items: T[]; next: string }

export type BatchResult = { done: string[]; failed: { id: string; reason: string }[] }

export type Brand = {
  id: string
  name: string
  initial: string
  logoUrl: string
  description: string
  visible: boolean
  sort: number
}

export type BrandInput = {
  name: string
  initial: string
  logo_url: string
  description: string
  visible: boolean
  sort: number
}

export type Category = {
  id: string
  parentId: string
  name: string
  iconUrl: string
  sort: number
  visible: boolean
  templateId: string
}

export type CategoryInput = {
  parent_id: string
  name: string
  icon_url: string
  sort: number
  visible: boolean
  attribute_template_id: string
}

export type Template = { id: string; name: string }

export type Attribute = {
  id: string
  templateId: string
  name: string
  kind: number
  options: string[]
  allowCustom: boolean
  sort: number
}

export type AttributeInput = {
  name: string
  kind: number
  options: string[]
  allow_custom: boolean
  sort: number
}

export type Recommendation = {
  id: string
  slot: string
  productId: string
  sort: number
  startAt: number
  endAt: number
  productName: string
}

export type RecommendationInput = {
  slot: string
  product_id: string
  sort: number
  start_at_unix: number
  end_at_unix: number
}

export type Stock = { skuId: string; onHand: number; reserved: number; available: number }

export type TrailRecord = {
  id: string
  productId: string
  action: string
  operator: string
  reason: string
  before: string
  after: string
  createdAt: number
}

export const trailActionText: Record<string, string> = {
  create: '新建',
  update: '编辑',
  submit: '提交审核',
  approve: '审核通过',
  reject: '驳回',
  publish: '上架',
  unpublish: '下架',
  delete: '删除',
  restore: '恢复',
  purge: '彻底删除',
  change_price: '改价',
}

export type OrderItem = { id: string; productName: string; specsJson: string; price: number; quantity: number; amount: number }

export type Order = {
  id: string
  orderNo: string
  status: number
  itemsAmount: number
  payAmount: number
  receiver: string
  phone: string
  address: string
  items: OrderItem[]
}

export const orderStatusText: Record<number, string> = {
  1: '待支付',
  2: '已支付',
  3: '已发货',
  4: '已完成',
  5: '已取消',
}

export type ReconcileDiff = { id: string; day: string; kind: string; orderId: string; paymentId: string; detail: string }

export type AdminUser = { id: string; username: string; displayName: string; role: string; status: number }

export const adminRoleText: Record<string, string> = { super: '超级管理员', product: '商品运营', order: '订单运营' }

function json(method: string, body: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}

function query(params: Record<string, string | number | string[] | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (Array.isArray(value)) value.forEach((item) => search.append(key, item))
    else if (value !== undefined && value !== '' && value !== 0) search.set(key, String(value))
  }
  const text = search.toString()
  return text ? `?${text}` : ''
}

function seg(id: string) {
  return encodeURIComponent(id)
}

async function body(p: Promise<unknown>): Promise<Row> {
  return asRow(await p) ?? {}
}

function page<T>(row: Row, key: string, read: (item: Row) => T): Page<T> {
  return { items: rows(row, key).map(read), next: str(row, 'next_page_token') }
}

function readCard(row: Row): ProductCard {
  return {
    id: str(row, 'id'),
    name: str(row, 'name'),
    mainImage: str(row, 'main_image'),
    minPrice: num(row, 'min_price'),
    salesCount: num(row, 'sales_count'),
    status: num(row, 'status'),
  }
}

export function parseSpecs(text: string): Record<string, string> {
  try {
    const value = asRow(JSON.parse(text) as unknown)
    if (!value) return {}
    const out: Record<string, string> = {}
    for (const [key, item] of Object.entries(value)) out[key] = typeof item === 'string' ? item : String(item)
    return out
  } catch {
    return {}
  }
}

export function specsLabel(text: string): string {
  const specs = parseSpecs(text)
  const parts = Object.entries(specs).map(([key, value]) => `${key}:${value}`)
  return parts.length ? parts.join(' ') : '默认'
}

function readSku(row: Row): Sku {
  const specsJson = str(row, 'specs_json')
  return {
    id: str(row, 'id'),
    specs: parseSpecs(specsJson),
    specsJson,
    price: num(row, 'price'),
    marketPrice: num(row, 'market_price'),
    image: str(row, 'image'),
    barcode: str(row, 'barcode'),
    enabled: flag(row, 'enabled'),
  }
}

function readProduct(row: Row): Product {
  return {
    id: str(row, 'id'),
    categoryId: str(row, 'category_id'),
    brandId: str(row, 'brand_id'),
    name: str(row, 'name'),
    subtitle: str(row, 'subtitle'),
    keywords: str(row, 'keywords'),
    unit: str(row, 'unit'),
    weightGram: num(row, 'weight_gram'),
    status: num(row, 'status'),
    minPrice: num(row, 'min_price'),
    maxPrice: num(row, 'max_price'),
    skus: rows(row, 'skus').map(readSku),
    images: strs(row, 'images'),
    detailHtml: str(row, 'detail_html'),
    params: rows(row, 'params').map((item) => ({ name: str(item, 'name'), value: str(item, 'value') })),
  }
}

function readBrand(row: Row): Brand {
  return {
    id: str(row, 'id'),
    name: str(row, 'name'),
    initial: str(row, 'initial'),
    logoUrl: str(row, 'logo_url'),
    description: str(row, 'description'),
    visible: flag(row, 'visible'),
    sort: num(row, 'sort'),
  }
}

// 分类可能是树也可能是平铺 统一拍平成带父级 id 的列表
function flattenCategories(list: Row[], parentId: string, out: Category[]) {
  for (const row of list) {
    const item: Category = {
      id: str(row, 'id'),
      parentId: str(row, 'parent_id') || parentId,
      name: str(row, 'name'),
      iconUrl: str(row, 'icon_url'),
      sort: num(row, 'sort'),
      visible: flag(row, 'visible'),
      templateId: str(row, 'attribute_template_id'),
    }
    if (item.id && !out.some((known) => known.id === item.id)) out.push(item)
    flattenCategories(rows(row, 'children'), item.id, out)
  }
}

function readAttribute(row: Row): Attribute {
  return {
    id: str(row, 'id'),
    templateId: str(row, 'template_id'),
    name: str(row, 'name'),
    kind: num(row, 'kind'),
    options: strs(row, 'options'),
    allowCustom: flag(row, 'allow_custom'),
    sort: num(row, 'sort'),
  }
}

function readRecommendation(row: Row): Recommendation {
  return {
    id: str(row, 'id'),
    slot: str(row, 'slot'),
    productId: str(row, 'product_id'),
    sort: num(row, 'sort'),
    startAt: num(row, 'start_at_unix'),
    endAt: num(row, 'end_at_unix'),
    productName: str(child(row, 'product'), 'name'),
  }
}

function readTrail(row: Row): TrailRecord {
  return {
    id: str(row, 'id'),
    productId: str(row, 'product_id'),
    action: str(row, 'action'),
    operator: str(row, 'operator'),
    reason: str(row, 'reason'),
    before: str(row, 'before'),
    after: str(row, 'after'),
    createdAt: num(row, 'created_at_unix'),
  }
}

function readOrder(row: Row): Order {
  return {
    id: str(row, 'id'),
    orderNo: str(row, 'order_no'),
    status: num(row, 'status'),
    itemsAmount: num(row, 'items_amount'),
    payAmount: num(row, 'pay_amount'),
    receiver: str(row, 'receiver'),
    phone: str(row, 'phone'),
    address: str(row, 'address'),
    items: rows(row, 'items').map((item) => ({
      id: str(item, 'id'),
      productName: str(item, 'product_name'),
      specsJson: str(item, 'specs_json'),
      price: num(item, 'price'),
      quantity: num(item, 'quantity'),
      amount: num(item, 'amount'),
    })),
  }
}

function readAdminUser(row: Row): AdminUser {
  return {
    id: str(row, 'id'),
    username: str(row, 'username'),
    displayName: str(row, 'display_name'),
    role: str(row, 'role'),
    status: num(row, 'status'),
  }
}

function readBatch(row: Row, key: string): BatchResult {
  return {
    done: strs(row, key),
    failed: rows(row, 'failed').map((item) => ({ id: str(item, 'id'), reason: str(item, 'reason') })),
  }
}

export function login(username: string, password: string) {
  return send('/v1/admin/login', json('POST', { username, password }))
}

export function readAccessToken(value: unknown): string {
  const row = asRow(value)
  return row ? str(row, 'access_token') : ''
}

export async function listProducts(pageToken = ''): Promise<Page<ProductCard>> {
  return page(await body(send(`/v1/admin/products${query({ page_size: 100, page_token: pageToken })}`)), 'products', readCard)
}

// 按页拉取直到没有下一页 后台商品量不大时足够
export async function listAllProducts(): Promise<ProductCard[]> {
  const all: ProductCard[] = []
  let token = ''
  for (let i = 0; i < 20; i++) {
    const next = await listProducts(token)
    all.push(...next.items)
    if (!next.next) break
    token = next.next
  }
  return all
}

export async function getProduct(id: string): Promise<Product> {
  return readProduct(child(await body(send(`/v1/admin/products/${seg(id)}`)), 'product'))
}

export async function createProduct(input: ProductInput): Promise<Product> {
  return readProduct(child(await body(send('/v1/admin/products', json('POST', input))), 'product'))
}

export async function updateProduct(id: string, input: ProductInput): Promise<Product> {
  return readProduct(child(await body(send(`/v1/admin/products/${seg(id)}`, json('PUT', { ...input, id }))), 'product'))
}

function transition(id: string, action: string, payload: Record<string, unknown> = {}) {
  return send(`/v1/admin/products/${seg(id)}:${action}`, json('POST', { ...payload, id }))
}

export function submitProduct(id: string) {
  return transition(id, 'submit')
}

export function approveProduct(id: string) {
  return transition(id, 'approve')
}

export function rejectProduct(id: string, reason: string) {
  return transition(id, 'reject', { reason })
}

export function publishProduct(id: string) {
  return transition(id, 'publish')
}

export function unpublishProduct(id: string) {
  return transition(id, 'unpublish')
}

export function deleteProduct(id: string) {
  return send(`/v1/admin/products/${seg(id)}`, { method: 'DELETE' })
}

export function restoreProduct(id: string) {
  return transition(id, 'restore')
}

export function purgeProduct(id: string) {
  return transition(id, 'purge')
}

export async function batchPublish(ids: string[]): Promise<BatchResult> {
  return readBatch(await body(send('/v1/admin/products:batchPublish', json('POST', { ids }))), 'published')
}

export async function batchUnpublish(ids: string[]): Promise<BatchResult> {
  return readBatch(await body(send('/v1/admin/products:batchUnpublish', json('POST', { ids }))), 'unpublished')
}

export function updateSkuPrice(productId: string, skuId: string, price: number) {
  return send(
    `/v1/admin/products/${seg(productId)}/skus/${seg(skuId)}:price`,
    json('POST', { product_id: productId, sku_id: skuId, price }),
  )
}

export async function listDeleted(pageToken = ''): Promise<Page<ProductCard>> {
  return page(await body(send(`/v1/admin/products:deleted${query({ page_size: 100, page_token: pageToken })}`)), 'products', readCard)
}

export async function listAudits(productId: string, pageToken = ''): Promise<Page<TrailRecord>> {
  const path = `/v1/admin/products/${seg(productId)}/audits${query({ page_size: 50, page_token: pageToken })}`
  return page(await body(send(path)), 'audits', readTrail)
}

export async function listLogs(productId: string, pageToken = ''): Promise<Page<TrailRecord>> {
  const path = `/v1/admin/product-logs${query({ product_id: productId, page_size: 50, page_token: pageToken })}`
  return page(await body(send(path)), 'logs', readTrail)
}

export async function listBrands(): Promise<Brand[]> {
  return rows(await body(send('/v1/admin/brands')), 'brands').map(readBrand)
}

export function createBrand(input: BrandInput) {
  return send('/v1/admin/brands', json('POST', input))
}

export function updateBrand(id: string, input: BrandInput) {
  return send(`/v1/admin/brands/${seg(id)}`, json('PATCH', { ...input, id }))
}

export function deleteBrand(id: string) {
  return send(`/v1/admin/brands/${seg(id)}`, { method: 'DELETE' })
}

export async function listCategories(): Promise<Category[]> {
  const out: Category[] = []
  flattenCategories(rows(await body(send('/v1/admin/categories')), 'categories'), '', out)
  return out
}

export function createCategory(input: CategoryInput) {
  return send('/v1/admin/categories', json('POST', input))
}

// 修改接口不接受父级 挪层级要删了重建
export function updateCategory(id: string, input: Omit<CategoryInput, 'parent_id'>) {
  return send(`/v1/admin/categories/${seg(id)}`, json('PATCH', { ...input, id }))
}

export function deleteCategory(id: string) {
  return send(`/v1/admin/categories/${seg(id)}`, { method: 'DELETE' })
}

export async function listTemplates(): Promise<Template[]> {
  return rows(await body(send('/v1/admin/attribute-templates')), 'templates').map((row) => ({ id: str(row, 'id'), name: str(row, 'name') }))
}

export function createTemplate(name: string) {
  return send('/v1/admin/attribute-templates', json('POST', { name }))
}

export function deleteTemplate(id: string) {
  return send(`/v1/admin/attribute-templates/${seg(id)}`, { method: 'DELETE' })
}

export async function listAttributes(templateId: string): Promise<Attribute[]> {
  const list = rows(await body(send(`/v1/admin/attribute-templates/${seg(templateId)}/attributes`)), 'attributes').map(readAttribute)
  return list.sort((a, b) => a.sort - b.sort)
}

export function createAttribute(templateId: string, input: AttributeInput) {
  return send(`/v1/admin/attribute-templates/${seg(templateId)}/attributes`, json('POST', { ...input, template_id: templateId }))
}

export function updateAttribute(id: string, input: AttributeInput) {
  return send(`/v1/admin/attributes/${seg(id)}`, json('PATCH', { ...input, id, clear_options: input.options.length === 0 }))
}

export function deleteAttribute(id: string) {
  return send(`/v1/admin/attributes/${seg(id)}`, { method: 'DELETE' })
}

export async function listRecommendations(): Promise<Recommendation[]> {
  return rows(await body(send('/v1/admin/recommendations')), 'recommendations').map(readRecommendation)
}

export function createRecommendation(input: RecommendationInput) {
  return send('/v1/admin/recommendations', json('POST', input))
}

export function updateRecommendation(id: string, input: RecommendationInput) {
  return send(`/v1/admin/recommendations/${seg(id)}`, json('PATCH', { ...input, id }))
}

export function deleteRecommendation(id: string) {
  return send(`/v1/admin/recommendations/${seg(id)}`, { method: 'DELETE' })
}

export async function getStocks(skuIds: string[]): Promise<Stock[]> {
  const list = rows(await body(send(`/v1/admin/stocks${query({ sku_ids: skuIds })}`)), 'stocks')
  return list.map((row) => ({
    skuId: str(row, 'sku_id'),
    onHand: num(row, 'on_hand'),
    reserved: num(row, 'reserved'),
    available: num(row, 'available'),
  }))
}

export function adjustStock(skuId: string, delta: number, reason: string) {
  return send('/v1/admin/stocks:adjust', json('POST', { sku_id: skuId, delta, reason }))
}

export async function listAdminOrders(status: number, pageToken = ''): Promise<Page<Order>> {
  return page(await body(send(`/v1/admin/orders${query({ status, page_size: 50, page_token: pageToken })}`)), 'orders', readOrder)
}

export function shipOrder(id: string) {
  return send(`/v1/admin/orders/${seg(id)}:ship`, json('POST', { id }))
}

export async function listDiffs(day: string, pageToken = ''): Promise<Page<ReconcileDiff>> {
  const path = `/v1/admin/payments/reconcile-diffs${query({ day, page_size: 100, page_token: pageToken })}`
  return page(await body(send(path)), 'diffs', (row) => ({
    id: str(row, 'id'),
    day: str(row, 'day'),
    kind: str(row, 'kind'),
    orderId: str(row, 'order_id'),
    paymentId: str(row, 'payment_id'),
    detail: str(row, 'detail'),
  }))
}

export async function getMe(): Promise<AdminUser> {
  return readAdminUser(child(await body(send('/v1/admin/me')), 'user'))
}

export async function listAdminUsers(): Promise<AdminUser[]> {
  return rows(await body(send('/v1/admin/users')), 'users').map(readAdminUser)
}

export function createAdminUser(input: { username: string; password: string; display_name: string; role: string }) {
  return send('/v1/admin/users', json('POST', input))
}

export function updateAdminUser(id: string, role: string, status: number) {
  return send(`/v1/admin/users/${seg(id)}`, json('PATCH', { id, role, status }))
}

export function resetAdminPassword(id: string, password: string) {
  return send(`/v1/admin/users/${seg(id)}:resetPassword`, json('POST', { id, password }))
}
