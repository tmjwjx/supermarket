import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { cancelOrder, confirmOrder, listOrders, orderStatus, readOrders, type Order } from '../api/order.ts'
import { createPayment, readPayment } from '../api/payment.ts'
import { formatPrice } from '../format/price.ts'

export default function OrdersPage() {
  const navigate = useNavigate()
  const [orders, setOrders] = useState<Order[]>([])
  const [ready, setReady] = useState(false)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  async function reload() {
    setOrders(readOrders(await listOrders()))
  }

  useEffect(() => {
    let gone = false
    reload()
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '订单加载失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [])

  async function onPay(order: Order) {
    setFailed(false)
    setMessage('')
    try {
      const payment = readPayment(await createPayment(order.id))
      if (!payment) {
        setFailed(true)
        setMessage('支付单没有建出来')
        return
      }
      navigate(`/pay/${encodeURIComponent(payment.id)}`)
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '去支付失败')
    }
  }

  async function onCancel(order: Order) {
    setFailed(false)
    setMessage('')
    try {
      await cancelOrder(order.id)
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '取消失败')
    }
  }

  async function onConfirm(order: Order) {
    setFailed(false)
    setMessage('')
    try {
      await confirmOrder(order.id)
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '确认收货失败')
    }
  }

  return (
    <section className="slip">
      <h1>我的订单</h1>
      <p className="lead">点订单看详情 完成的订单可以评价</p>
      {ready && orders.length === 0 && !message ? <p className="note">还没有订单 <Link to="/">去逛逛</Link></p> : null}
      <ul className="skus">
        {orders.map((order) => (
          <li key={order.id}>
            <Link className="row" to={`/orders/${encodeURIComponent(order.id)}`}>
              <strong>{order.orderNo || order.id}</strong>
              <span>{orderStatus(order.status)}</span>
              <span className="price">{formatPrice(order.payAmount)}</span>
              <span>{order.lines.map((line) => `${line.name || '商品'} x${line.quantity}`).join(' ')}</span>
              <span className="meta">查看详情</span>
            </Link>
            {order.status === 1 ? (
              <div className="actions">
                <button className="primary" type="button" onClick={() => void onPay(order)}>
                  去支付
                </button>
                <button className="quiet" type="button" onClick={() => void onCancel(order)}>
                  取消
                </button>
              </div>
            ) : null}
            {order.status === 3 ? (
              <button className="primary" type="button" onClick={() => void onConfirm(order)}>
                确认收货
              </button>
            ) : null}
          </li>
        ))}
      </ul>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
