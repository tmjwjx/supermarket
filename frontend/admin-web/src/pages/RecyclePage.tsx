import { useState } from 'react'
import { listDeleted, purgeProduct, restoreProduct, type Page, type ProductCard } from '../api/admin.ts'
import { formatCents } from '../money.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

export default function RecyclePage() {
  const { data, setData, error, loading, reload } = useLoad<Page<ProductCard>>(() => listDeleted(), { items: [], next: '' })
  const [confirming, setConfirming] = useState('')
  const [busy, setBusy] = useState(false)
  const { note, ok, fail } = useNote()

  async function run(action: () => Promise<unknown>, done: string) {
    setBusy(true)
    try {
      await action()
      ok(done)
      reload()
    } catch (err) {
      fail(err, '操作失败')
    } finally {
      setBusy(false)
      setConfirming('')
    }
  }

  async function more() {
    try {
      const next = await listDeleted(data.next)
      setData({ items: [...data.items, ...next.items], next: next.next })
    } catch (err) {
      fail(err, '加载失败')
    }
  }

  return (
    <section>
      <h1>回收站</h1>
      <p className="hint">恢复后商品回到已下架 彻底删除会连同规格图片和详情一起清除且无法找回</p>
      <Notice text={error} tone="error" />
      <Notice {...note} />
      {!loading && data.items.length === 0 ? <p>回收站是空的</p> : null}
      {data.items.length > 0 ? (
        <table>
          <thead>
            <tr><th>名称</th><th>最低价</th><th>操作</th></tr>
          </thead>
          <tbody>
            {data.items.map((item) => (
              <tr key={item.id}>
                <td>{item.name || '未命名'}<div className="hint">{item.id}</div></td>
                <td>{formatCents(item.minPrice)}</td>
                <td>
                  <div className="actions">
                    <button type="button" disabled={busy} onClick={() => void run(() => restoreProduct(item.id), '已恢复为下架')}>恢复</button>
                    {confirming === item.id ? (
                      <>
                        <span>确定彻底删除 {item.name} 吗</span>
                        <button type="button" className="danger" disabled={busy} onClick={() => void run(() => purgeProduct(item.id), '已彻底删除')}>确认彻底删除</button>
                        <button type="button" onClick={() => setConfirming('')}>取消</button>
                      </>
                    ) : (
                      <button type="button" className="danger" disabled={busy} onClick={() => setConfirming(item.id)}>彻底删除</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {data.next ? <button type="button" onClick={() => void more()}>加载更多</button> : null}
    </section>
  )
}
