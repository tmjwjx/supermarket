import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { clearBrowseHistories, listBrowseHistories, readBrowseHistories, type BrowseItem } from '../api/browse.ts'
import { formatPrice } from '../format/price.ts'

export default function BrowsePage() {
  const [items, setItems] = useState<BrowseItem[]>([])
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let gone = false
    listBrowseHistories()
      .then((body) => {
        if (!gone) setItems(readBrowseHistories(body))
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '浏览记录加载失败')
      })
    return () => {
      gone = true
    }
  }, [])

  async function onClear() {
    setMessage('')
    setFailed(false)
    try {
      await clearBrowseHistories()
      setItems([])
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '清空失败')
    }
  }

  return (
    <section>
      <div className="slip">
        <h1>浏览记录</h1>
        <p className="lead">最近看过的商品</p>
        {items.length > 0 ? (
          <button className="quiet" type="button" onClick={onClear}>
            清空
          </button>
        ) : null}
      </div>
      {items.length === 0 && !message ? <p className="note">还没有浏览记录</p> : null}
      <div className="goods">
        {items.map((item) => (
          <Link className="item" key={item.productId} to={`/products/${encodeURIComponent(item.productId)}`}>
            <strong>{item.product?.name || '未命名商品'}</strong>
            {item.product ? <span className="price">{formatPrice(item.product.minPrice)}</span> : null}
          </Link>
        ))}
      </div>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
