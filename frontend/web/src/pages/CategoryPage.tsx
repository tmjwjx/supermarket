import { useEffect, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { GoodsList, Pager } from '../components/Goods.tsx'
import { usePager } from '../components/usePager.ts'
import { categoryTree, listBrands, listCategories, readBrands, readCategories, type Brand, type Category } from '../api/catalog.ts'
import { listProducts, readProductPage, type ProductCard } from '../api/product.ts'

const sorts = [
  { id: '', label: '综合' },
  { id: 'latest', label: '最新' },
  { id: 'price', label: '价格升' },
  { id: 'price_desc', label: '价格降' },
  { id: 'sales', label: '销量' },
]

export default function CategoryPage() {
  const { id = '' } = useParams()
  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [brandId, setBrandId] = useState('')
  const [minText, setMinText] = useState('')
  const [maxText, setMaxText] = useState('')
  const [range, setRange] = useState({ min: 0, max: 0 })
  const [sort, setSort] = useState('')
  const [products, setProducts] = useState<ProductCard[]>([])
  const [nextToken, setNextToken] = useState('')
  const [ready, setReady] = useState(false)
  const [message, setMessage] = useState('')
  const [failed, setFailed] = useState(false)
  const pager = usePager(`${id}|${brandId}|${range.min}|${range.max}|${sort}`)

  useEffect(() => {
    let gone = false
    listCategories()
      .then((body) => {
        if (!gone) setCategories(readCategories(body))
      })
      .catch(() => {})
    listBrands()
      .then((body) => {
        if (!gone) setBrands(readBrands(body))
      })
      .catch(() => {})
    return () => {
      gone = true
    }
  }, [])

  useEffect(() => {
    if (!id) return
    let gone = false
    setReady(false)
    setMessage('')
    setFailed(false)
    listProducts({
      categoryId: id,
      brandId,
      minPrice: range.min,
      maxPrice: range.max,
      orderBy: sort,
      pageSize: 20,
      pageToken: pager.token,
    })
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
        setMessage(err instanceof Error ? err.message : '商品加载失败')
      })
      .finally(() => {
        if (!gone) setReady(true)
      })
    return () => {
      gone = true
    }
  }, [id, brandId, range.min, range.max, sort, pager.token])

  function onRange(event: FormEvent) {
    event.preventDefault()
    const min = toCents(minText)
    const max = toCents(maxText)
    if (min === null || max === null) {
      setFailed(true)
      setMessage('价格按元填 最多两位小数')
      return
    }
    if (min > 0 && max > 0 && min > max) {
      setFailed(true)
      setMessage('最低价不能高于最高价')
      return
    }
    setRange({ min, max })
  }

  function clearRange() {
    setMinText('')
    setMaxText('')
    setRange({ min: 0, max: 0 })
  }

  const tree = categoryTree(categories)
  const current = categories.find((item) => item.id === id)

  return (
    <section className="shelf">
      <aside className="slip side">
        <h2>分类</h2>
        <ul className="tree">
          {tree.map(({ root, children }) => (
            <li key={root.id}>
              <Link className={root.id === id ? 'on' : ''} to={`/category/${encodeURIComponent(root.id)}`}>
                {root.name}
              </Link>
              {children.length ? (
                <ul>
                  {children.map((child) => (
                    <li key={child.id}>
                      <Link className={child.id === id ? 'on' : ''} to={`/category/${encodeURIComponent(child.id)}`}>
                        {child.name}
                      </Link>
                    </li>
                  ))}
                </ul>
              ) : null}
            </li>
          ))}
        </ul>
      </aside>
      <div>
        <div className="slip">
          <h1>{current?.name || '分类商品'}</h1>
          <label>
            品牌
            <select value={brandId} onChange={(event) => setBrandId(event.target.value)}>
              <option value="">全部品牌</option>
              {brands.map((brand) => (
                <option key={brand.id} value={brand.id}>
                  {brand.name}
                </option>
              ))}
            </select>
          </label>
          <form className="range" onSubmit={onRange}>
            <label>
              最低价 元
              <input value={minText} onChange={(event) => setMinText(event.target.value)} inputMode="decimal" placeholder="不限" />
            </label>
            <label>
              最高价 元
              <input value={maxText} onChange={(event) => setMaxText(event.target.value)} inputMode="decimal" placeholder="不限" />
            </label>
            <button className="quiet" type="submit">
              筛选
            </button>
            <button className="quiet" type="button" onClick={clearRange}>
              清空
            </button>
          </form>
          <div className="sorts">
            {sorts.map((item) => (
              <button key={item.id} className={item.id === sort ? 'chip on' : 'chip'} type="button" onClick={() => setSort(item.id)}>
                {item.label}
              </button>
            ))}
          </div>
        </div>
        {ready && products.length === 0 && !message ? <p className="note">这里暂时没有商品</p> : null}
        <GoodsList items={products} />
        <Pager page={pager.page} hasNext={nextToken !== ''} onPrev={pager.prev} onNext={() => pager.next(nextToken)} />
        {message ? <p className={failed ? 'note bad' : 'note'}>{message}</p> : null}
      </div>
    </section>
  )
}

// 元转分 空串算不限 格式不对返回 null
function toCents(text: string): number | null {
  const value = text.trim()
  if (!value) return 0
  if (!/^\d+(\.\d{1,2})?$/.test(value)) return null
  return Math.round(Number(value) * 100)
}
