import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { listAddresses, readAddresses, type Address } from '../api/address.ts'
import { listCartItems, readCartItems, removeCartItems, updateCartItem, type CartItem } from '../api/cart.ts'
import { createOrder, readOrder } from '../api/order.ts'
import { createPayment, readPayment } from '../api/payment.ts'
import { getProduct, readProduct, type Sku } from '../api/product.ts'
import { formatPrice } from '../format/price.ts'
import { specsText } from '../format/specs.ts'

export default function CartPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [params] = useSearchParams()
  const buySku = params.get('buy') ?? ''
  const buyQty = Math.max(1, Math.trunc(Number(params.get('qty') ?? 1)) || 1)
  const buyProduct = params.get('product') ?? ''
  // 每种结算入口各用一个幂等号 避免立即购买和购物车结算撞上同一张订单
  const request = useRef({ search: '', id: '' })
  const [items, setItems] = useState<CartItem[]>([])
  const [buying, setBuying] = useState<{ name: string; sku: Sku | null } | null>(null)
  const [ready, setReady] = useState(false)
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const [addresses, setAddresses] = useState<Address[]>([])
  const [addressId, setAddressId] = useState('')
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)

  async function reload() {
    const [cartBody, addressBody] = await Promise.all([listCartItems(), listAddresses()])
    const nextItems = readCartItems(cartBody)
    const nextAddresses = readAddresses(addressBody)
    setItems(nextItems)
    setAddresses(nextAddresses)
    setAddressId((current) =>
      nextAddresses.some((item) => item.id === current)
        ? current
        : nextAddresses.find((item) => item.isDefault)?.id || nextAddresses[0]?.id || '',
    )
  }

  useEffect(() => {
    let gone = false
    reload()
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '购物车加载失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [])

  useEffect(() => {
    setBuying(null)
    if (!buySku || !buyProduct) return
    let gone = false
    getProduct(buyProduct)
      .then((body) => {
        if (gone) return
        const product = readProduct(body)
        setBuying({ name: product?.name ?? '', sku: product?.skus.find((sku) => sku.id === buySku) ?? null })
      })
      .catch(() => {})
    return () => {
      gone = true
    }
  }, [buySku, buyProduct])

  async function onCheck(item: CartItem, checked: boolean) {
    setFailed(false)
    setMessage('')
    try {
      await updateCartItem({ skuId: item.skuId, quantity: item.quantity, checked })
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '没改成')
    }
  }

  async function onQuantity(item: CartItem) {
    const quantity = Number(drafts[item.skuId] ?? item.quantity)
    if (!Number.isInteger(quantity) || quantity < 1) {
      setFailed(true)
      setMessage('数量得是大于 0 的整数')
      return
    }
    setFailed(false)
    setMessage('')
    try {
      await updateCartItem({ skuId: item.skuId, quantity, checked: item.checked })
      setDrafts((current) => {
        const next = { ...current }
        delete next[item.skuId]
        return next
      })
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '数量没改成')
    }
  }

  async function onRemove(skuIds: string[]) {
    if (skuIds.length === 0) return
    setFailed(false)
    setMessage('')
    try {
      await removeCartItems(skuIds)
      await reload()
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '没删掉')
    }
  }

  async function onCheckout() {
    const lines = buySku
      ? [{ skuId: buySku, quantity: buyQty }]
      : items
          .filter((item) => item.checked && !item.invalid && item.quantity > 0)
          .map((item) => ({ skuId: item.skuId, quantity: item.quantity }))
    if (!addressId) {
      setFailed(true)
      setMessage('请先选好收货地址')
      return
    }
    if (lines.length === 0) {
      setFailed(true)
      setMessage('先勾选要结算的商品')
      return
    }
    setFailed(false)
    setMessage('')
    if (!request.current.id || request.current.search !== location.search) {
      request.current = { search: location.search, id: crypto.randomUUID() }
    }
    try {
      const order = readOrder(await createOrder({ lines, addressId, requestId: request.current.id }))
      if (!order) {
        setFailed(true)
        setMessage('订单没有建出来')
        return
      }
      const payment = readPayment(await createPayment(order.id))
      if (!payment) {
        setFailed(true)
        setMessage('支付单没有建出来')
        return
      }
      navigate(`/pay/${encodeURIComponent(payment.id)}`)
    } catch (err) {
      setFailed(true)
      setMessage(err instanceof Error ? err.message : '下单失败')
    }
  }

  const amount = buySku
    ? (buying?.sku?.price ?? 0) * buyQty
    : items.filter((item) => item.checked && !item.invalid).reduce((sum, item) => sum + item.price * item.quantity, 0)
  const manage = `/addresses?from=${encodeURIComponent(`${location.pathname}${location.search}`)}`

  return (
    <section className="slip">
      <h1>{buySku ? '立即购买' : '购物车'}</h1>
      <p className="lead">{buySku ? '只结算这一件 不动购物车' : '勾选后就能下单'}</p>
      {buySku ? (
        <>
          <ul className="skus">
            <li>
              <strong>{buying?.name || '商品'}</strong>
              <span>{specsText(buying?.sku?.specs ?? '', '默认规格')}</span>
              {buying?.sku ? <span className="price">{formatPrice(buying.sku.price)}</span> : null}
              <span className="meta">数量 {buyQty}</span>
            </li>
          </ul>
          <p className="switch">
            <Link to="/cart">回到购物车</Link>
          </p>
        </>
      ) : (
        <>
          {ready && items.length === 0 && !message ? <p className="note">车子是空的 <Link to="/">去逛逛</Link></p> : null}
          <ul className="skus">
            {items.map((item) => (
              <li key={item.skuId}>
                <label className="check">
                  <input
                    type="checkbox"
                    checked={item.checked}
                    disabled={item.invalid}
                    onChange={(event) => void onCheck(item, event.target.checked)}
                  />
                  {item.name || item.skuId}
                  {item.invalid ? ' 已失效' : item.shortStock ? ' 库存不足' : ''}
                </label>
                <span>{specsText(item.specs)}</span>
                <span className="price">{formatPrice(item.price)}</span>
                <label>
                  数量
                  <input
                    value={drafts[item.skuId] ?? String(item.quantity)}
                    onChange={(event) => setDrafts((current) => ({ ...current, [item.skuId]: event.target.value }))}
                    inputMode="numeric"
                  />
                </label>
                <button className="quiet" type="button" onClick={() => void onQuantity(item)}>
                  更新数量
                </button>
                <button className="quiet" type="button" onClick={() => void onRemove([item.skuId])}>
                  删除
                </button>
              </li>
            ))}
          </ul>
          {items.some((item) => item.invalid) ? (
            <button className="quiet" type="button" onClick={() => void onRemove(items.filter((item) => item.invalid).map((item) => item.skuId))}>
              清空失效商品
            </button>
          ) : null}
        </>
      )}
      {addresses.length > 0 ? (
        <label>
          收货地址
          <select value={addressId} onChange={(event) => setAddressId(event.target.value)}>
            {addresses.map((item) => (
              <option key={item.id} value={item.id}>
                {item.receiver} {item.phone} {item.province}
                {item.city}
                {item.district}
                {item.detail}
                {item.isDefault ? ' 默认' : ''}
              </option>
            ))}
          </select>
        </label>
      ) : ready ? (
        <p className="note">还没有收货地址</p>
      ) : null}
      <p className="switch">
        <Link to={manage}>{addresses.length > 0 ? '管理收货地址' : '去添加收货地址'}</Link>
      </p>
      <p className="price">合计 {formatPrice(amount)}</p>
      <button className="primary" type="button" onClick={() => void onCheckout()}>
        提交订单
      </button>
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
