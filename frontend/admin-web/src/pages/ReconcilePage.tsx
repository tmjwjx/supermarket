import { useState } from 'react'
import { listDiffs, type Page, type ReconcileDiff } from '../api/admin.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function ReconcilePage() {
  const [day, setDay] = useState('')
  const [input, setInput] = useState('')
  const diffs = useLoad<Page<ReconcileDiff>>(() => listDiffs(day), { items: [], next: '' }, [day])
  const { note, fail } = useNote()

  async function more() {
    try {
      const next = await listDiffs(day, diffs.data.next)
      diffs.setData({ items: [...diffs.data.items, ...next.items], next: next.next })
    } catch (err) {
      fail(err, '加载失败')
    }
  }

  return (
    <section>
      <h1>对账差异</h1>
      <form
        className="row"
        onSubmit={(event) => {
          event.preventDefault()
          setDay(input)
        }}
      >
        <label>日期 留空看全部<input type="date" value={input} onChange={(e) => setInput(e.target.value)} /></label>
        <button type="submit">查询</button>
        <button type="button" onClick={() => diffs.reload()}>刷新</button>
      </form>
      <Notice text={diffs.error} tone="error" />
      <Notice {...note} />
      {!diffs.loading && diffs.data.items.length === 0 ? <p>没有差异</p> : null}
      {diffs.data.items.length > 0 ? (
        <table>
          <thead><tr><th>日期</th><th>类型</th><th>订单</th><th>支付单</th><th>说明</th></tr></thead>
          <tbody>
            {diffs.data.items.map((d) => (
              <tr key={d.id}>
                <td>{d.day}</td>
                <td>{d.kind}</td>
                <td>{d.orderId || '-'}</td>
                <td>{d.paymentId || '-'}</td>
                <td>{d.detail}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {diffs.data.next ? <button type="button" onClick={() => void more()}>加载更多</button> : null}
    </section>
  )
}
