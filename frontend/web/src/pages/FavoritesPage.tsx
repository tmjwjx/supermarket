import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listFavorites, readFavorites, removeFavorite, type Favorite } from '../api/favorite.ts'
import { formatPrice } from '../format/price.ts'

export default function FavoritesPage() {
  const [items, setItems] = useState<Favorite[]>([])
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let gone = false
    listFavorites()
      .then((body) => {
        if (!gone) setItems(readFavorites(body))
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '收藏加载失败')
      })
    return () => {
      gone = true
    }
  }, [])

  async function onRemove(productId: string) {
    setMessage('')
    setFailed(false)
    try {
      await removeFavorite(productId)
      setItems((current) => current.filter((item) => item.productId !== productId))
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '取消收藏失败')
    }
  }

  return (
    <section>
      <div className="slip">
        <h1>我的收藏</h1>
        <p className="lead">你留下来的商品</p>
      </div>
      {items.length === 0 && !message ? <p className="note">还没有收藏</p> : null}
      <div className="goods">
        {items.map((item) => (
          <div className="item" key={item.productId}>
            <Link to={`/products/${encodeURIComponent(item.productId)}`}>
              <strong>{item.invalid ? '已失效' : item.product?.name || '未命名商品'}</strong>
              {item.product ? <span className="price">{formatPrice(item.product.minPrice)}</span> : null}
            </Link>
            <button className="quiet" type="button" onClick={() => onRemove(item.productId)}>
              取消收藏
            </button>
          </div>
        ))}
      </div>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
