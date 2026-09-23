import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { addCartItem } from '../api/cart.ts'
import { getStocks, listReviews, readReviews, readStocks, type Review } from '../api/catalog.ts'
import { addFavorite, listFavorites, readFavorites, removeFavorite } from '../api/favorite.ts'
import { getProduct, getProductDetail, readProduct, readProductDetail, statusOff, statusOnSale, type Product } from '../api/product.ts'
import { sanitizeHtml } from '../format/html.ts'
import { formatPrice } from '../format/price.ts'
import { specsMap, specsText, stockText } from '../format/specs.ts'
import { loadToken } from '../session/index.ts'

export default function ProductPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [product, setProduct] = useState<Product | null>(null)
  const [skuId, setSkuId] = useState('')
  const [stocks, setStocks] = useState<Record<string, number> | null>(null)
  const [quantity, setQuantity] = useState(1)
  const [detail, setDetail] = useState('')
  const [favorited, setFavorited] = useState(false)
  const [reviews, setReviews] = useState<Review[]>([])
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)
  const signedIn = loadToken() !== ''

  useEffect(() => {
    if (!id) return
    let gone = false
    setProduct(null)
    setStocks(null)
    setDetail('')
    setMessage('')
    setFailed(false)
    getProduct(id)
      .then((body) => {
        if (gone) return
        const next = readProduct(body)
        if (!next) {
          setFailed(true)
          setMessage('没有找到这件商品')
          return
        }
        setProduct(next)
        setDetail((current) => current || next.detailHtml)
        const first = next.skus.find((sku) => sku.enabled) ?? next.skus[0]
        setSkuId(first?.id ?? '')
        setQuantity(1)
        const ids = next.skus.map((sku) => sku.id)
        if (ids.length === 0) return
        getStocks(ids)
          .then((stockBody) => {
            if (gone) return
            const map: Record<string, number> = {}
            for (const sku of next.skus) map[sku.id] = 0
            for (const row of readStocks(stockBody)) map[row.skuId] = row.available
            setStocks(map)
          })
          .catch(() => {})
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '商品加载失败')
      })
    getProductDetail(id)
      .then((body) => {
        const html = readProductDetail(body)
        if (!gone && html) setDetail(html)
      })
      .catch(() => {})
    listReviews(id)
      .then((body) => {
        if (!gone) setReviews(readReviews(body))
      })
      .catch(() => {})
    if (signedIn) {
      listFavorites()
        .then((body) => {
          if (!gone) setFavorited(readFavorites(body).some((item) => item.productId === id))
        })
        .catch(() => {})
    }
    return () => {
      gone = true
    }
  }, [id, signedIn])

  const safeDetail = useMemo(() => sanitizeHtml(detail), [detail])
  const selected = product?.skus.find((sku) => sku.id === skuId) ?? null
  const hero = selected?.image || product?.image || ''
  const offShelf = product?.status === statusOff
  const available = selected && stocks ? stocks[selected.id] : undefined
  const soldOut = available !== undefined && available <= 0
  const limit = available !== undefined && available > 0 ? available : Infinity
  const onSale = product?.status === statusOnSale
  const blocked = !selected || !selected.enabled || !onSale || soldOut
  const qty = Math.max(1, Math.min(quantity, limit))

  function goLogin() {
    navigate(`/login?from=${encodeURIComponent(`/products/${id}`)}`)
  }

  function pick(next: string) {
    setSkuId(next)
    setQuantity(1)
  }

  const chosen = selected ? specsMap(selected.specs) : {}

  // 换一个规格值时 优先保留其他已选的值 凑不成组合就落到第一个带这个值的 SKU
  function pickValue(name: string, value: string) {
    if (!product) return
    const want = { ...chosen, [name]: value }
    const rows = product.skus.map((sku) => ({ sku, specs: specsMap(sku.specs) }))
    const exact = rows.find((row) => Object.entries(want).every(([k, v]) => row.specs[k] === v))
    const loose = rows.find((row) => row.specs[name] === value)
    const hit = exact ?? loose
    if (hit) pick(hit.sku.id)
  }

  function changeQuantity(raw: number) {
    const n = Number.isFinite(raw) ? Math.trunc(raw) : 1
    setQuantity(Math.max(1, Math.min(n, limit)))
  }

  async function onAdd() {
    if (!selected || blocked) return
    if (!signedIn) {
      goLogin()
      return
    }
    setMessage('')
    setFailed(false)
    try {
      await addCartItem({ skuId: selected.id, quantity: qty })
      setMessage('已放入购物车')
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '没放进去')
    }
  }

  function onBuy() {
    if (!selected || blocked) return
    const params = new URLSearchParams({ buy: selected.id, qty: String(qty), product: id })
    navigate(`/cart?${params.toString()}`)
  }

  async function onFavorite() {
    if (!signedIn) {
      goLogin()
      return
    }
    setMessage('')
    setFailed(false)
    try {
      if (favorited) {
        await removeFavorite(id)
        setFavorited(false)
        setMessage('已取消收藏')
      } else {
        await addFavorite(id)
        setFavorited(true)
        setMessage('已收藏')
      }
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '收藏没改成')
    }
  }

  return (
    <section className="slip">
      <p className="switch">
        <Link to="/">返回货架</Link>
      </p>
      {hero ? <img className="hero" src={hero} alt="" /> : null}
      <h1>{product?.name || '商品详情'}</h1>
      {product?.subtitle ? <p className="lead">{product.subtitle}</p> : null}
      {offShelf ? <p className="tag off">已下架</p> : null}
      {selected ? (
        <p className="price">
          {formatPrice(selected.price)}
          {selected.marketPrice > selected.price ? <s className="meta"> {formatPrice(selected.marketPrice)}</s> : null}
        </p>
      ) : null}
      {selected && !offShelf && available !== undefined ? (
        <p className={soldOut ? 'note bad' : 'note'}>{stockText(available)}</p>
      ) : null}
      {product?.specOptions.length ? (
        product.specOptions.map((option) => (
          <div className="specs" key={option.name}>
            <span className="meta">{option.name}</span>
            {option.values.map((value) => (
              <button
                key={value}
                className={chosen[option.name] === value ? 'chip on' : 'chip'}
                type="button"
                onClick={() => pickValue(option.name, value)}
              >
                {value}
              </button>
            ))}
          </div>
        ))
      ) : product?.skus.length ? (
        <div className="specs">
          {product.skus.map((sku) => (
            <button
              key={sku.id}
              className={sku.id === skuId ? 'chip on' : 'chip'}
              type="button"
              disabled={!sku.enabled}
              onClick={() => pick(sku.id)}
            >
              {specsText(sku.specs, '默认规格')}
              {sku.enabled ? '' : ' 暂不可售'}
            </button>
          ))}
        </div>
      ) : product ? (
        <p className="note">还没有可买的规格</p>
      ) : null}
      {product ? (
        <>
          <label>
            数量
            <input
              type="number"
              min={1}
              max={Number.isFinite(limit) ? limit : undefined}
              value={qty}
              disabled={blocked}
              onChange={(event) => changeQuantity(Number(event.target.value))}
            />
          </label>
          <div className="actions">
            <button className="primary" type="button" disabled={blocked} onClick={() => void onAdd()}>
              {offShelf ? '已下架' : !onSale ? '暂不可售' : soldOut ? '缺货' : '加入购物车'}
            </button>
            <button className="primary alt" type="button" disabled={blocked} onClick={onBuy}>
              立即购买
            </button>
          </div>
          <button className="quiet" type="button" onClick={() => void onFavorite()}>
            {!signedIn ? '登录后收藏' : favorited ? '取消收藏' : '收藏'}
          </button>
        </>
      ) : null}
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      {product?.params.length ? (
        <>
          <h2>商品参数</h2>
          <ul>
            {product.params.map((param) => (
              <li key={param.name}>
                <span className="meta">{param.name}</span> {param.value}
              </li>
            ))}
          </ul>
        </>
      ) : null}
      <h2>图文详情</h2>
      {safeDetail ? <div className="detail" dangerouslySetInnerHTML={{ __html: safeDetail }} /> : <p className="note">暂无图文详情</p>}
      <h2>评价</h2>
      {reviews.length === 0 ? <p className="note">还没有评价</p> : null}
      <ul>
        {reviews.map((item) => (
          <li key={item.id}>
            <strong>{item.displayName}</strong> {item.stars} 星 {item.specs ? `规格 ${specsText(item.specs)}` : ''}
            <p>{item.content}</p>
          </li>
        ))}
      </ul>
      <p className="meta">买过的商品可以在 <Link to="/orders">我的订单</Link> 里评价</p>
    </section>
  )
}
