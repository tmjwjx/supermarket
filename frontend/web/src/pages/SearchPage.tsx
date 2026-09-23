import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { GoodsList, Pager } from '../components/Goods.tsx'
import { usePager } from '../components/usePager.ts'
import { readProductPage, searchProducts, type ProductCard } from '../api/product.ts'

const sorts = [
  { id: '', label: '综合' },
  { id: 'latest', label: '最新' },
  { id: 'price', label: '价格升' },
  { id: 'price_desc', label: '价格降' },
  { id: 'sales', label: '销量' },
]

export default function SearchPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const q = (params.get('q') ?? '').trim()
  const [draft, setDraft] = useState(q)
  const [sort, setSort] = useState('')
  const [products, setProducts] = useState<ProductCard[]>([])
  const [nextToken, setNextToken] = useState('')
  const [ready, setReady] = useState(false)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)
  const [seen, setSeen] = useState(q)
  const pager = usePager(`${q}|${sort}`)
  if (seen !== q) {
    setSeen(q)
    setDraft(q)
  }

  useEffect(() => {
    if (!q) {
      setProducts([])
      setNextToken('')
      setReady(true)
      return
    }
    let gone = false
    setReady(false)
    setMessage('')
    setFailed(false)
    searchProducts(q, pager.token, 20, sort)
      .then((body) => {
        if (gone) return
        const page = readProductPage(body)
        setProducts(page.products)
        setNextToken(page.nextPageToken)
      })
      .catch((err: unknown) => {
        if (gone) return
        setProducts([])
        setNextToken('')
        setFailed(true)
        setMessage(err instanceof Error ? err.message : '搜索失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [q, sort, pager.token])

  return (
    <section>
      <div className="slip">
        <form
          className="search"
          onSubmit={(event) => {
            event.preventDefault()
            const next = draft.trim()
            if (next) navigate(`/search?q=${encodeURIComponent(next)}`)
          }}
        >
          <input value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="搜索商品" aria-label="搜索商品" />
          <button className="primary" type="submit">
            搜索
          </button>
        </form>
        <h1>{q ? `搜索 ${q}` : '搜索商品'}</h1>
        <div className="sorts">
          {sorts.map((item) => (
            <button key={item.id} className={item.id === sort ? 'chip on' : 'chip'} type="button" onClick={() => setSort(item.id)}>
              {item.label}
            </button>
          ))}
        </div>
      </div>
      {!q ? <p className="note">输入关键词找找看</p> : null}
      {q && ready && products.length === 0 && !message ? <p className="note">没搜到相关商品</p> : null}
      <GoodsList items={products} />
      <Pager page={pager.page} hasNext={nextToken !== ''} onPrev={pager.prev} onNext={() => pager.next(nextToken)} />
      {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
    </section>
  )
}