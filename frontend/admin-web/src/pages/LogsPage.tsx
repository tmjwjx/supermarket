import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listLogs, trailActionText, type Page, type TrailRecord } from '../api/admin.ts'
import { formatTime, useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function LogsPage() {
  const [params, setParams] = useSearchParams()
  const productId = params.get('product_id') ?? ''
  const [input, setInput] = useState(productId)
  const logs = useLoad<Page<TrailRecord>>(() => listLogs(productId), { items: [], next: '' }, [productId])
  const { note, fail } = useNote()

  async function more() {
    try {
      const next = await listLogs(productId, logs.data.next)
      logs.setData({ items: [...logs.data.items, ...next.items], next: next.next })
    } catch (err) {
      fail(err, '加载失败')
    }
  }

  return (
    <section>
      <h1>操作日志</h1>
      <form
        className="row"
        onSubmit={(event) => {
          event.preventDefault()
          setParams(input.trim() ? { product_id: input.trim() } : {})
        }}
      >
        <label>商品 id 留空看全部<input value={input} onChange={(e) => setInput(e.target.value)} /></label>
        <button type="submit">查询</button>
        <button type="button" onClick={() => logs.reload()}>刷新</button>
      </form>
      <Notice text={logs.error} tone="error" />
      <Notice {...note} />
      {!logs.loading && logs.data.items.length === 0 ? <p>没有日志</p> : null}
      {logs.data.items.length > 0 ? (
        <table>
          <thead><tr><th>时间</th><th>商品</th><th>动作</th><th>操作人</th><th>变更前</th><th>变更后</th></tr></thead>
          <tbody>
            {logs.data.items.map((log) => (
              <tr key={log.id}>
                <td>{formatTime(log.createdAt)}</td>
                <td><Link to={`/products/${encodeURIComponent(log.productId)}`}>{log.productId}</Link></td>
                <td>{trailActionText[log.action] ?? log.action}</td>
                <td>{log.operator || '-'}</td>
                <td className="hint">{log.before || '-'}</td>
                <td>{log.after || log.reason || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {logs.data.next ? <button type="button" onClick={() => void more()}>加载更多</button> : null}
    </section>
  )
}
