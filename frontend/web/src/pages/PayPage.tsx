import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getOrder, readOrder } from '../api/order.ts'
import { getPayment, paymentStatus, readPayment, simulatePayment, type Payment } from '../api/payment.ts'
import { formatPrice } from '../format/price.ts'

export default function PayPage() {
  const { id = '' } = useParams()
  const [payment, setPayment] = useState<Payment | null>(null)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!id) return
    let gone = false
    getPayment(id)
      .then((body) => {
        if (gone) return
        const next = readPayment(body)
        if (!next) {
          setFailed(true)
          setMessage('没有这张支付单')
          return
        }
        setPayment(next)
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '支付单加载失败')
      })
    return () => {
      gone = true
    }
  }, [id])

  async function onSimulate(success: boolean) {
    if (!id) return
    setFailed(false)
    setMessage('')
    try {
      const next = readPayment(await simulatePayment(id, success))
      if (next) setPayment(next)
      const orderId = next?.orderId || payment?.orderId || ''
      if (orderId) {
        const order = readOrder(await getOrder(orderId))
        if (order?.status === 5) {
          setFailed(false)
          setMessage('订单已取消 款项已退回')
          return
        }
      }
      setFailed(!success)
      setMessage(success ? '模拟支付成功' : '模拟支付失败')
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '模拟支付没完成')
    }
  }

  return (
    <section className="slip">
      <h1>模拟支付</h1>
      <p className="lead">{payment ? paymentStatus(payment.status) : '正在读取支付单'}</p>
      {payment ? (
        <>
          <p className="price">{formatPrice(payment.amount)}</p>
          {payment.orderId ? <p className="note">订单 {payment.orderId}</p> : null}
        </>
      ) : null}
      <div className="actions">
        <button className="primary" type="button" onClick={() => void onSimulate(true)}>
          支付成功
        </button>
        <button className="primary alt" type="button" onClick={() => void onSimulate(false)}>
          支付失败
        </button>
      </div>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      <p className="switch">
        <Link to="/orders">查看订单</Link>
      </p>
    </section>
  )
}
