import { useState } from 'react'
import { Link } from 'react-router-dom'
import { listAdminOrders, orderStatusText, shipOrder, specsLabel, type Order, type Page } from '../api/admin.ts'
import { formatCents } from '../money.ts'
import { useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

const paid = 2

export default function OrdersPage() {
  const [status, setStatus] = useState(0)
  const orders = useLoad<Page<Order>>(() => listAdminOrders(status), { items: [], next: '' }, [status])
  const [busy, setBusy] = useState(false)
  const { note, ok, fail } = useNote()

  async function ship(order: Order) {
    if (!window.confirm(`确认订单 ${order.orderNo || order.id} 已发货`)) return
    setBusy(true)
    try {
      await shipOrder(order.id)
      ok(`订单 ${order.orderNo || order.id} 已发货`)
      orders.reload()
    } catch (err) {
      fail(err, '发货失败')
    } finally {
      setBusy(false)
    }
  }

  async function more() {
    try {
      const next = await listAdminOrders(status, orders.data.next)
      orders.setData({ items: [...orders.data.items, ...next.items], next: next.next })
    } catch (err) {
      fail(err, '加载失败')
    }
  }

  return (
    <section>
      <h1>订单</h1>
      <div className="row">
        <label>
          状态
          <select value={status} onChange={(e) => setStatus(Number(e.target.value))}>
            <option value={0}>全部</option>
            {Object.entries(orderStatusText).map(([value, text]) => <option key={value} value={value}>{text}</option>)}
          </select>
        </label>
        <button type="button" onClick={() => orders.reload()}>刷新</button>
        <Link to="/ship">按单号发货</Link>
      </div>
      <Notice text={orders.error} tone="error" />
      <Notice {...note} />
      {!orders.loading && orders.data.items.length === 0 ? <p>没有订单</p> : null}
      {orders.data.items.length > 0 ? (
        <table>
          <thead><tr><th>订单号</th><th>状态</th><th>实付</th><th>收货</th><th>商品</th><th>操作</th></tr></thead>
          <tbody>
            {orders.data.items.map((o) => (
              <tr key={o.id}>
                <td>{o.orderNo || o.id}<div className="hint">{o.id}</div></td>
                <td><span className={`tag ${o.status === paid ? 'on' : ''}`}>{orderStatusText[o.status] ?? o.status}</span></td>
                <td>{formatCents(o.payAmount)}</td>
                <td>{o.receiver} {o.phone}<div className="hint">{o.address}</div></td>
                <td>
                  {o.items.map((item) => (
                    <div key={item.id}>{item.productName} {specsLabel(item.specsJson)} × {item.quantity} {formatCents(item.amount)}</div>
                  ))}
                </td>
                <td>{o.status === paid ? <button type="button" disabled={busy} onClick={() => void ship(o)}>发货</button> : null}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {orders.data.next ? <button type="button" onClick={() => void more()}>加载更多</button> : null}
    </section>
  )
}
