import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { categoryTree, listCategories, listRecommendations, readCategories, readRecommendations, type Category, type Recommendation } from '../api/catalog.ts'
import { listProducts, readProductPage, type ProductCard } from '../api/product.ts'
import { GoodsList, Pager } from '../components/Goods.tsx'
import { usePager } from '../components/usePager.ts'
import { formatPrice } from '../format/price.ts'

const slots = [
  { id: 'new', label: '新品' },
  { id: 'hot', label: '热卖' },
  { id: 'topic', label: '专题' },
]

export default function HomePage() {
  const navigate = useNavigate()
  const [products, setProducts] = useState<ProductCard[]>([])
  const [nextToken, setNextToken] = useState('')
  const [categories, setCategories] = useState<Category[]>([])
  const [picks, setPicks] = useState<Recommendation[]>([])
  const [query, setQuery] = useState('')
  const [ready, setReady] = useState(false)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)
  const pager = usePager('home')

  useEffect(() => {
    let gone = false
    listCategories()
      .then((body) => {
        if (!gone) setCategories(readCategories(body))
      })
      .catch(() => {})
    Promise.all(slots.map((slot) => listRecommendations(slot.id).then(readRecommendations).catch(() => [])))
      .then((groups) => {
        if (!gone) setPicks(groups.flat())
      })
      .catch(() => {})
    return () => {
      gone = true
    }
  }, [])

  useEffect(() => {
    let gone = false
    setReady(false)
    listProducts({ pageSize: 20, pageToken: pager.token })
      .then((body) => {
        if (gone) return
        const page = readProductPage(body)
        setProducts(page.products)
        setNextToken(page.nextPageToken)
      })
      .catch((err: unknown) => {
        if (gone) return
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '商品加载失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [pager.token])

  const roots = categoryTree(categories).map((node) => node.root)

  return (
    <section>
      <div className="slip">
        <form
          className="search"
          onSubmit={(event) => {
            event.preventDefault()
            const q = query.trim()
            if (q) navigate(`/search?q=${encodeURIComponent(q)}`)
          }}
        >
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索商品" aria-label="搜索商品" />
          <button className="primary" type="submit">
            搜索
          </button>
        </form>
        <h1>今日货架</h1>
        <p className="lead">街口正在卖的东西</p>
        {roots.length ? (
          <div className="sorts">
            {roots.map((item) => (
              <Link key={item.id} className="chip" to={`/category/${encodeURIComponent(item.id)}`}>
                {item.name}
              </Link>
            ))}
          </div>
        ) : null}
      </div>
      {picks.length ? (
        <div className="slip">
          <h2>推荐</h2>
          <div className="goods">
            {picks.map((item) => (
              <Link className="item" key={item.id} to={`/products/${encodeURIComponent(item.productId)}`}>
                {item.image ? <img className="thumb" src={item.image} alt="" /> : <span className="thumb empty">无图</span>}
                <span>
                  <strong>{item.name || '推荐商品'}</strong>
                  <span className="price">{formatPrice(item.minPrice)}</span>
                </span>
              </Link>
            ))}
          </div>
        </div>
      ) : null}
      {ready && products.length === 0 && !message ? <p className="note">货架上还没有商品</p> : null}
      <GoodsList items={products} />
      <Pager page={pager.page} hasNext={nextToken !== ''} onPrev={pager.prev} onNext={() => pager.next(nextToken)} />
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}
