import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  approveProduct,
  batchPublish,
  batchUnpublish,
  deleteProduct,
  listAllProducts,
  ProductStatus,
  productStatusText,
  publishProduct,
  rejectProduct,
  submitProduct,
  unpublishProduct,
  type BatchResult,
  type ProductCard,
} from '../api/admin.ts'
import { formatCents } from '../money.ts'
import { useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function ProductsPage() {
  const [items, setItems] = useState<ProductCard[]>([])
  const [loaded, setLoaded] = useState(false)
  const [filter, setFilter] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [rejecting, setRejecting] = useState('')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const { note, ok, fail } = useNote()

  async function refresh() {
    try {
      setItems(await listAllProducts())
      setLoaded(true)
    } catch (err) {
      fail(err, '商品加载失败')
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    try {
      await action()
      ok(done)
      await refresh()
    } catch (err) {
      fail(err, '操作失败')
    } finally {
      setBusy(false)
    }
  }

  async function runBatch(action: (ids: string[]) => Promise<BatchResult>, verb: string) {
    const ids = [...selected]
    if (ids.length === 0) {
      fail(null, '先勾选商品')
      return
    }
    setBusy(true)
    try {
      const result = await action(ids)
      const names = new Map(items.map((item) => [item.id, item.name]))
      const failed = result.failed.map((f) => `${names.get(f.id) || f.id} ${f.reason}`).join(' / ')
      const text = `${verb}成功 ${result.done.length} 个` + (result.failed.length ? ` 失败 ${result.failed.length} 个 ${failed}` : '')
      if (result.failed.length) fail(new Error(text), text)
      else ok(text)
      setSelected(new Set())
      await refresh()
    } catch (err) {
      fail(err, `批量${verb}失败`)
    } finally {
      setBusy(false)
    }
  }

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const shown = items.filter(
    (item) => (filter === 0 || item.status === filter) && (!keyword.trim() || item.name.includes(keyword.trim())),
  )
  const allChecked = shown.length > 0 && shown.every((item) => selected.has(item.id))

  function toggleAll() {
    setSelected(allChecked ? new Set() : new Set(shown.map((item) => item.id)))
  }

  function actions(item: ProductCard) {
    const s = item.status
    const out = [<Link key="edit" to={`/products/${encodeURIComponent(item.id)}`}>编辑</Link>]
    if (s === ProductStatus.Draft || s === ProductStatus.Rejected) {
      out.push(<button key="submit" type="button" disabled={busy} onClick={() => void run(() => submitProduct(item.id), '已提交审核')}>提交审核</button>)
    }
    if (s === ProductStatus.Pending) {
      out.push(<button key="approve" type="button" disabled={busy} onClick={() => void run(() => approveProduct(item.id), '已通过')}>通过</button>)
      out.push(
        <button key="reject" type="button" disabled={busy} onClick={() => { setRejecting(item.id); setReason('') }}>驳回</button>,
      )
    }
    if (s === ProductStatus.Approved || s === ProductStatus.Off) {
      out.push(<button key="publish" type="button" disabled={busy} onClick={() => void run(() => publishProduct(item.id), '已上架')}>上架</button>)
    }
    if (s === ProductStatus.OnSale) {
      out.push(<button key="unpublish" type="button" disabled={busy} onClick={() => void run(() => unpublishProduct(item.id), '已下架')}>下架</button>)
    }
    out.push(
      <button
        key="delete"
        type="button"
        className="danger"
        disabled={busy}
        onClick={() => {
          if (window.confirm(`把 ${item.name || item.id} 移入回收站`)) void run(() => deleteProduct(item.id), '已移入回收站')
        }}
      >
        删除
      </button>,
    )
    out.push(<Link key="stock" to={`/stocks?product=${encodeURIComponent(item.id)}`}>库存</Link>)
    out.push(<Link key="logs" to={`/logs?product_id=${encodeURIComponent(item.id)}`}>日志</Link>)
    return out
  }

  return (
    <section>
      <h1>商品</h1>
      <div className="row">
        <Link to="/products/new">新建商品</Link>
        <label>
          状态
          <select value={filter} onChange={(event) => setFilter(Number(event.target.value))}>
            <option value={0}>全部</option>
            {Object.entries(productStatusText).map(([value, text]) => <option key={value} value={value}>{text}</option>)}
          </select>
        </label>
        <label>
          名称
          <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="按名称筛选" />
        </label>
        <button type="button" disabled={busy} onClick={() => void runBatch(batchPublish, '上架')}>批量上架</button>
        <button type="button" disabled={busy} onClick={() => void runBatch(batchUnpublish, '下架')}>批量下架</button>
        <button type="button" disabled={busy} onClick={() => void refresh()}>刷新</button>
      </div>
      <p className="hint">已勾选 {selected.size} 个 只有已通过或已下架能上架 只有在售能下架</p>
      <Notice {...note} />
      {loaded && shown.length === 0 ? <p>没有商品</p> : null}
      {shown.length > 0 ? (
        <table>
          <thead>
            <tr>
              <th><input type="checkbox" checked={allChecked} onChange={toggleAll} /></th>
              <th>名称</th>
              <th>状态</th>
              <th>最低价</th>
              <th>销量</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {shown.map((item) => (
              <tr key={item.id}>
                <td><input type="checkbox" checked={selected.has(item.id)} onChange={() => toggle(item.id)} /></td>
                <td>
                  <Link to={`/products/${encodeURIComponent(item.id)}`}>{item.name || '未命名'}</Link>
                  <div className="hint">{item.id}</div>
                </td>
                <td><span className={`tag ${item.status === ProductStatus.OnSale ? 'on' : ''}`}>{productStatusText[item.status] ?? item.status}</span></td>
                <td>{formatCents(item.minPrice)}</td>
                <td>{item.salesCount}</td>
                <td>
                  <div className="actions">{actions(item)}</div>
                  {rejecting === item.id ? (
                    <div className="actions">
                      <input value={reason} onChange={(event) => setReason(event.target.value)} placeholder="驳回理由 可不填" />
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => {
                          setRejecting('')
                          void run(() => rejectProduct(item.id, reason.trim() || '未通过审核'), '已驳回')
                        }}
                      >
                        确认驳回
                      </button>
                      <button type="button" onClick={() => setRejecting('')}>取消</button>
                    </div>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  )
}
