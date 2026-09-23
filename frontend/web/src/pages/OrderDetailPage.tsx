import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { cancelOrder, confirmOrder, getOrder, orderCompleted, orderStatus, readOrder, type Order } from '../api/order.ts'
import { createPayment, readPayment } from '../api/payment.ts'
import { formatPrice } from '../format/price.ts'
import { specsText } from '../format/specs.ts'

export default function OrderDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const notice = (location.state as { notice?: unknown } | null)?.notice
  const [order, setOrder] = useState<Order | null>(null)
  const [message, setMessage] = useState(typeof notice === 'string' ? notice : '')
  const [failed, setFailed] = useState(false)

  async function reload() {
    const next = readOrder(await getOrder(id))
    if (!next) throw new Error('没有找到这张订单')
    setOrder(next)
  }

  useEffect(() => {
    if (!id) return
    let gone = false
    getOrder(id)
      .then((body) => {
        if (gone) return
        const next = readOrder(body)
        if (!next) {
          setFailed(true)
          setMessage('没有找到这张订单')
          return
        }
        setOrder(next)
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '订单加载失败')
      })
    return () => {
      gone = true
    }
  }, [id])

  async function act(task: () => Promise<unknown>, fail: string) {
    setFailed(false)
    setMessage('')
    try {
      await task()
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : fail)
    }
  }

  async function onPay() {
    setFailed(false)
    setMessage('')
    try {
      const payment = readPayment(await createPayment(id))
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

  const completed = order?.status === orderCompleted

  return (
    <section className="slip">
      <p className="switch">
        <Link to="/orders">返回我的订单</Link>
      </p>
      <h1>订单详情</h1>
      {order ? (
        <>
          <dl className="card">
            <dt>订单号</dt>
            <dd>{order.orderNo || order.id}</dd>
            <dt>状态</dt>
            <dd>{orderStatus(order.status)}</dd>
            <dt>商品金额</dt>
            <dd>{formatPrice(order.itemsAmount)}</dd>
            <dt>实付</dt>
            <dd className="price">{formatPrice(order.payAmount)}</dd>
            <dt>收货人</dt>
            <dd>
              {order.receiver} {order.phone}
            </dd>
            <dt>收货地址</dt>
            <dd>{order.address || '未填写'}</dd>
            {order.status === 1 && order.expiresAt > 0 ? (
              <>
                <dt>支付截止</dt>
                <dd>{new Date(order.expiresAt * 1000).toLocaleString('zh-CN')}</dd>
              </>
            ) : null}
          </dl>
          <h2>订单项</h2>
          <ul className="skus">
            {order.lines.map((line, index) => (
              <li key={line.id || `${line.skuId}-${index}`} className="line">
                {line.image ? <img className="thumb" src={line.image} alt="" /> : <span className="thumb empty">无图</span>}
                <span>
                  {line.productId ? (
                    <Link to={`/products/${encodeURIComponent(line.productId)}`}>
                      <strong>{line.name || '商品'}</strong>
                    </Link>
                  ) : (
                    <strong>{line.name || '商品'}</strong>
                  )}
                  <span className="meta">{specsText(line.specs, '默认规格')}</span>
                  <span className="meta">
                    {formatPrice(line.price)} x{line.quantity}
                  </span>
                  <span className="price">{formatPrice(line.amount || line.price * line.quantity)}</span>
                  {completed && line.id && line.productId ? (
                    line.reviewed ? (
                      <span className="tag">已评价</span>
                    ) : (
                      <Link className="tag go" to={`/orders/${encodeURIComponent(order.id)}/review/${encodeURIComponent(line.id)}`}>
                        评价
                      </Link>
                    )
                  ) : null}
                </span>
              </li>
            ))}
          </ul>
          {order.status === 1 ? (
            <div className="actions">
              <button className="primary" type="button" onClick={() => void onPay()}>
                去支付
              </button>
              <button className="quiet" type="button" onClick={() => void act(() => cancelOrder(order.id), '取消失败')}>
                取消订单
              </button>
            </div>
          ) : null}
          {order.status === 3 ? (
            <button className="primary" type="button" onClick={() => void act(() => confirmOrder(order.id), '确认收货失败')}>
              确认收货
            </button>
          ) : null}
        </>
      ) : null}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
