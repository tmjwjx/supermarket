import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { adjustStock, getProduct, getStocks, listAllProducts, specsLabel, type ProductCard, type Stock } from '../api/admin.ts'
import { formatCents } from '../money.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

type Line = { skuId: string; label: string }

function splitIds(text: string): string[] {
  return [...new Set(text.split(/[\s,，]+/).map((s) => s.trim()).filter(Boolean))]
}

export default function StockPage() {
  const [params, setParams] = useSearchParams()
  const products = useLoad<ProductCard[]>(listAllProducts, [])
  const [productId, setProductId] = useState(params.get('product') ?? '')
  const [idsText, setIdsText] = useState('')
  const [lines, setLines] = useState<Line[]>([])
  const [stocks, setStocks] = useState<Map<string, Stock>>(new Map())
  const [qty, setQty] = useState<Record<string, string>>({})
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const { note, ok, fail, clear } = useNote()

  async function query(next: Line[]) {
    setLines(next)
    if (next.length === 0) {
      setStocks(new Map())
      return
    }
    const list = await getStocks(next.map((l) => l.skuId))
    setStocks(new Map(list.map((s) => [s.skuId, s])))
  }

  async function fromProduct(id: string) {
    clear()
    setProductId(id)
    setParams(id ? { product: id } : {}, { replace: true })
    if (!id) return
    setBusy(true)
    try {
      const p = await getProduct(id)
      const next = p.skus.map((sku) => ({ skuId: sku.id, label: `${specsLabel(sku.specsJson)} ${formatCents(sku.price)}${sku.enabled ? '' : ' 已停用'}` }))
      setIdsText(next.map((l) => l.skuId).join('\n'))
      await query(next)
      if (next.length === 0) ok('该商品没有 SKU')
    } catch (err) {
      fail(err, '读取商品失败')
    } finally {
      setBusy(false)
    }
  }

  useEffect(() => {
    const initial = params.get('product')
    if (initial) void fromProduct(initial)
    // 只在进入页面时按地址栏带出一次
  }, [])

  async function fromIds() {
    clear()
    const ids = splitIds(idsText)
    if (ids.length === 0) return fail(null, '先填 SKU id')
    setBusy(true)
    try {
      const labels = new Map(lines.map((l) => [l.skuId, l.label]))
      await query(ids.map((skuId) => ({ skuId, label: labels.get(skuId) ?? '' })))
    } catch (err) {
      fail(err, '查询库存失败')
    } finally {
      setBusy(false)
    }
  }

  async function adjust(skuId: string, sign: 1 | -1) {
    clear()
    const n = Number(qty[skuId] ?? '')
    if (!Number.isInteger(n) || n <= 0) return fail(null, '数量要填正整数')
    const current = stocks.get(skuId)
    if (sign < 0 && current && n > current.available) return fail(null, `可用只有 ${current.available} 出库数量不能更多`)
    setBusy(true)
    try {
      await adjustStock(skuId, sign * n, reason.trim() || (sign > 0 ? '入库' : '出库'))
      setQty((prev) => ({ ...prev, [skuId]: '' }))
      await query(lines)
      ok(`${sign > 0 ? '入库' : '出库'} ${n} 件完成`)
    } catch (err) {
      fail(err, '调整失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section>
      <h1>库存</h1>
      <Notice text={products.error} tone="error" />
      <div className="row">
        <label>
          从商品带出 SKU
          <select value={productId} disabled={busy} onChange={(e) => void fromProduct(e.target.value)}>
            <option value="">不选</option>
            {products.data.map((p) => <option key={p.id} value={p.id}>{p.name || p.id}</option>)}
          </select>
        </label>
      </div>
      <label>
        或直接填 SKU id 每行一个或用逗号分隔
        <textarea value={idsText} onChange={(e) => setIdsText(e.target.value)} style={{ minHeight: '4rem' }} />
      </label>
      <div className="row">
        <button type="button" disabled={busy} onClick={() => void fromIds()}>查询库存</button>
        <label>调整原因 可空<input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="采购入库 盘点出库" /></label>
      </div>
      <Notice {...note} />
      {lines.length > 0 ? (
        <table>
          <thead>
            <tr><th>SKU</th><th>实物库存</th><th>已预占</th><th>可用</th><th>入库 出库</th></tr>
          </thead>
          <tbody>
            {lines.map((line) => {
              const s = stocks.get(line.skuId)
              return (
                <tr key={line.skuId}>
                  <td>{line.label || line.skuId}{line.label ? <div className="hint">{line.skuId}</div> : null}</td>
                  <td>{s ? s.onHand : '-'}</td>
                  <td>{s ? s.reserved : '-'}</td>
                  <td>{s ? <strong>{s.available}</strong> : '无库存记录'}</td>
                  <td>
                    <div className="actions">
                      <input value={qty[line.skuId] ?? ''} onChange={(e) => setQty((prev) => ({ ...prev, [line.skuId]: e.target.value }))} placeholder="数量" inputMode="numeric" style={{ width: '6rem' }} />
                      <button type="button" disabled={busy} onClick={() => void adjust(line.skuId, 1)}>入库</button>
                      <button type="button" disabled={busy} onClick={() => void adjust(line.skuId, -1)}>出库</button>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      ) : null}
    </section>
  )
}
