import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  approveProduct,
  listAllProducts,
  listAudits,
  ProductStatus,
  rejectProduct,
  trailActionText,
  type Page,
  type ProductCard,
  type TrailRecord,
} from '../api/admin.ts'
import { formatCents } from '../money.ts'
import { formatTime, useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function AuditPage() {
  const [params, setParams] = useSearchParams()
  const lookup = params.get('id') ?? ''
  const pending = useLoad<ProductCard[]>(() => listAllProducts().then((all) => all.filter((p) => p.status === ProductStatus.Pending)), [])
  const audits = useLoad<Page<TrailRecord>>(() => (lookup ? listAudits(lookup) : Promise.resolve({ items: [], next: '' })), { items: [], next: '' }, [lookup])
  const [reasons, setReasons] = useState<Record<string, string>>({})
  const [input, setInput] = useState(lookup)
  const [busy, setBusy] = useState(false)
  const { note, ok, fail } = useNote()

  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    try {
      await action()
      ok(done)
      pending.reload()
      audits.reload()
    } catch (err) {
      fail(err, '操作失败')
    } finally {
      setBusy(false)
    }
  }

  async function more() {
    try {
      const next = await listAudits(lookup, audits.data.next)
      audits.setData({ items: [...audits.data.items, ...next.items], next: next.next })
    } catch (err) {
      fail(err, '加载失败')
    }
  }

  return (
    <section>
      <h1>审核</h1>
      <Notice text={pending.error} tone="error" />
      <Notice {...note} />
      <h2>待审核 {pending.data.length}</h2>
      {!pending.loading && pending.data.length === 0 ? <p>没有待审核的商品</p> : null}
      {pending.data.length > 0 ? (
        <table>
          <thead><tr><th>名称</th><th>最低价</th><th>操作</th></tr></thead>
          <tbody>
            {pending.data.map((item) => (
              <tr key={item.id}>
                <td><Link to={`/products/${encodeURIComponent(item.id)}`}>{item.name || '未命名'}</Link><div className="hint">{item.id}</div></td>
                <td>{formatCents(item.minPrice)}</td>
                <td>
                  <div className="actions">
                    <button type="button" disabled={busy} onClick={() => void run(() => approveProduct(item.id), `${item.name} 已通过`)}>通过</button>
                    <input
                      value={reasons[item.id] ?? ''}
                      onChange={(e) => setReasons((prev) => ({ ...prev, [item.id]: e.target.value }))}
                      placeholder="驳回理由 可不填"
                    />
                    <button
                      type="button"
                      className="danger"
                      disabled={busy}
                      onClick={() => void run(() => rejectProduct(item.id, (reasons[item.id] ?? '').trim() || '未通过审核'), `${item.name} 已驳回`)}
                    >
                      驳回
                    </button>
                    <button type="button" onClick={() => { setInput(item.id); setParams({ id: item.id }) }}>审核记录</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}

      <h2>审核记录</h2>
      <form
        className="row"
        onSubmit={(event) => {
          event.preventDefault()
          setParams(input.trim() ? { id: input.trim() } : {})
        }}
      >
        <label>商品 id<input value={input} onChange={(e) => setInput(e.target.value)} /></label>
        <button type="submit">查看</button>
      </form>
      <Notice text={audits.error} tone="error" />
      {lookup && !audits.loading && audits.data.items.length === 0 ? <p>没有审核记录</p> : null}
      {audits.data.items.length > 0 ? (
        <table>
          <thead><tr><th>时间</th><th>动作</th><th>操作人</th><th>理由</th></tr></thead>
          <tbody>
            {audits.data.items.map((a) => (
              <tr key={a.id}>
                <td>{formatTime(a.createdAt)}</td>
                <td>{trailActionText[a.action] ?? a.action}</td>
                <td>{a.operator || '-'}</td>
                <td>{a.reason || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {audits.data.next ? <button type="button" onClick={() => void more()}>加载更多</button> : null}
    </section>
  )
}
