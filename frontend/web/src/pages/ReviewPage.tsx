import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { createReview } from '../api/catalog.ts'
import { getOrder, orderCompleted, readOrder, type Order, type OrderLine } from '../api/order.ts'
import { specsText } from '../format/specs.ts'

// 订单项和商品编号都从订单里取 用户只填星级和内容
export default function ReviewPage() {
  const { id = '', itemId = '' } = useParams()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [line, setLine] = useState<OrderLine | null>(null)
  const [stars, setStars] = useState(5)
  const [content, setContent] = useState('')
  const [anonymous, setAnonymous] = useState(false)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!id) return
    let gone = false
    getOrder(id)
      .then((body) => {
        if (gone) return
        const next = readOrder(body)
        setOrder(next)
        const found = next?.lines.find((item) => item.id === itemId) ?? null
        setLine(found)
        if (!found) {
          setFailed(true)
          setMessage('这张订单里没有这件商品')
        }
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '订单加载失败')
      })
    return () => {
      gone = true
    }
  }, [id, itemId])

  const back = `/orders/${encodeURIComponent(id)}`
  const allowed = order?.status === orderCompleted && line !== null && !line.reviewed && line.productId !== ''

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (!line || !allowed) return
    const text = content.trim()
    if (!text) {
      setFailed(true)
      setMessage('写几句评价吧')
      return
    }
    setBusy(true)
    setFailed(false)
    setMessage('')
    try {
      await createReview({ orderItemId: line.id, productId: line.productId, stars, content: text, anonymous })
      navigate(back, { replace: true, state: { notice: '评价已提交' } })
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '评价没写上')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="slip">
      <p className="switch">
        <Link to={back}>返回订单详情</Link>
      </p>
      <h1>写评价</h1>
      {line ? (
        <p className="lead">
          {line.name || '商品'} {specsText(line.specs)}
        </p>
      ) : null}
      {order && line && order.status !== orderCompleted ? <p className="note">订单完成后才能评价</p> : null}
      {line?.reviewed ? <p className="note">这件商品已评价</p> : null}
      {allowed ? (
        <form onSubmit={(event) => void onSubmit(event)}>
          <label>
            星级
            <select value={stars} onChange={(event) => setStars(Number(event.target.value))}>
              {[5, 4, 3, 2, 1].map((n) => (
                <option key={n} value={n}>
                  {n} 星
                </option>
              ))}
            </select>
          </label>
          <label>
            内容
            <textarea value={content} onChange={(event) => setContent(event.target.value)} rows={4} maxLength={500} required />
          </label>
          <label className="check">
            <input type="checkbox" checked={anonymous} onChange={(event) => setAnonymous(event.target.checked)} /> 匿名
          </label>
          <button className="primary" type="submit" disabled={busy}>
            提交评价
          </button>
        </form>
      ) : null}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
