import { Link } from 'react-router-dom'
import type { ProductCard } from '../api/product.ts'
import { formatPrice } from '../format/price.ts'

export function GoodsList({ items }: { items: ProductCard[] }) {
  return (
    <div className="goods">
      {items.map((item) => (
        <Link className="item" key={item.id} to={`/products/${encodeURIComponent(item.id)}`}>
          {item.image ? <img className="thumb" src={item.image} alt="" /> : <span className="thumb empty">无图</span>}
          <span>
            <strong>{item.name || '未命名商品'}</strong>
            <span className="price">
              {formatPrice(item.minPrice)}
              {item.marketPrice > item.minPrice ? <s className="meta"> {formatPrice(item.marketPrice)}</s> : null}
            </span>
            <span className="meta">已售 {item.sales}</span>
          </span>
        </Link>
      ))}
    </div>
  )
}

export function Pager({ page, hasNext, onPrev, onNext }: { page: number; hasNext: boolean; onPrev: () => void; onNext: () => void }) {
  if (page <= 1 && !hasNext) return null
  return (
    <div className="pager">
      <button className="quiet" type="button" disabled={page <= 1} onClick={onPrev}>
        上一页
      </button>
      <span className="meta">第 {page} 页</span>
      <button className="quiet" type="button" disabled={!hasNext} onClick={onNext}>
        下一页
      </button>
    </div>
  )
}
