import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  AttributeKind,
  createProduct,
  getProduct,
  listAttributes,
  listBrands,
  listCategories,
  ProductStatus,
  productStatusText,
  submitProduct,
  updateProduct,
  updateSkuPrice,
  type Attribute,
  type Brand,
  type Category,
  type Product,
  type ProductInput,
  type ProductParam,
  type SkuInput,
} from '../api/admin.ts'
import { centsToYuan, yuanToCents } from '../money.ts'
import { errText, useNote } from './shared.ts'
import Notice from './Notice.tsx'

type SkuDraft = {
  key: number
  id: string
  specs: Record<string, string>
  specsJson: string
  price: string
  marketPrice: string
  image: string
  barcode: string
  enabled: boolean
  origPrice: number
}

type Base = {
  name: string
  subtitle: string
  keywords: string
  unit: string
  weightGram: string
  categoryId: string
  brandId: string
}

const steps = ['基本信息', '规格与价格', '参数', '图片与详情']
const maxImages = 9
const emptyBase: Base = { name: '', subtitle: '', keywords: '', unit: '', weightGram: '', categoryId: '', brandId: '' }

let skuSeq = 0

function blankSku(): SkuDraft {
  return { key: ++skuSeq, id: '', specs: {}, specsJson: '', price: '', marketPrice: '', image: '', barcode: '', enabled: true, origPrice: 0 }
}

function specKeys(skus: SkuDraft[]): string[] {
  const keys: string[] = []
  for (const sku of skus) for (const key of Object.keys(sku.specs)) if (!keys.includes(key)) keys.push(key)
  return keys
}

export default function ProductEditPage() {
  const { id = 'new' } = useParams()
  const isNew = id === 'new'
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [status, setStatus] = useState(0)
  const [base, setBase] = useState<Base>(emptyBase)
  const [freeDims, setFreeDims] = useState<string[]>([])
  const [newDim, setNewDim] = useState('')
  const [skus, setSkus] = useState<SkuDraft[]>([blankSku()])
  const [params, setParams] = useState<ProductParam[]>([])
  const [images, setImages] = useState<string[]>([''])
  const [detailHtml, setDetailHtml] = useState('')
  const [categories, setCategories] = useState<Category[]>([])
  const [brands, setBrands] = useState<Brand[]>([])
  const [attrs, setAttrs] = useState<Attribute[]>([])
  const [attrsError, setAttrsError] = useState('')
  const [attrsFor, setAttrsFor] = useState('')
  const attrSeq = useRef(0)
  const { note, ok, fail, clear } = useNote()

  function fill(p: Product) {
    setStatus(p.status)
    setBase({
      name: p.name,
      subtitle: p.subtitle,
      keywords: p.keywords,
      unit: p.unit,
      weightGram: p.weightGram ? String(p.weightGram) : '',
      categoryId: p.categoryId,
      brandId: p.brandId,
    })
    const drafts = p.skus.map((sku) => ({
      key: ++skuSeq,
      id: sku.id,
      specs: sku.specs,
      specsJson: sku.specsJson,
      price: centsToYuan(sku.price),
      marketPrice: sku.marketPrice ? centsToYuan(sku.marketPrice) : '',
      image: sku.image,
      barcode: sku.barcode,
      enabled: sku.enabled,
      origPrice: sku.price,
    }))
    setSkus(drafts.length ? drafts : [blankSku()])
    setFreeDims(specKeys(drafts))
    setParams(p.params)
    setImages(p.images.length ? p.images : [''])
    setDetailHtml(p.detailHtml)
  }

  useEffect(() => {
    let gone = false
    setLoading(true)
    Promise.all([listCategories(), listBrands(), isNew ? Promise.resolve(null) : getProduct(id)])
      .then(([cats, brandList, product]) => {
        if (gone) return
        setCategories(cats)
        setBrands(brandList)
        if (product) fill(product)
      })
      .catch((err: unknown) => {
        if (!gone) fail(err, '加载失败')
      })
      .finally(() => {
        if (!gone) setLoading(false)
      })
    return () => {
      gone = true
    }
  }, [id, isNew, fail])

  const category = categories.find((c) => c.id === base.categoryId)
  const templateId = category?.templateId ?? ''

  useEffect(() => {
    const seq = ++attrSeq.current
    setAttrsError('')
    if (!templateId) {
      setAttrs([])
      return
    }
    listAttributes(templateId)
      .then((list) => {
        if (seq !== attrSeq.current) return
        setAttrs(list)
        setAttrsFor(templateId)
      })
      .catch((err: unknown) => {
        if (seq === attrSeq.current) {
          setAttrs([])
          setAttrsError(errText(err, '模板属性加载失败'))
        }
      })
  }, [templateId])

  const locked = status === ProductStatus.OnSale
  const hasTemplate = templateId !== ''
  const specAttrs = useMemo(() => attrs.filter((a) => a.kind === AttributeKind.Spec), [attrs])
  const paramAttrs = useMemo(() => attrs.filter((a) => a.kind === AttributeKind.Param), [attrs])
  const specAttrMap = new Map(specAttrs.map((a) => [a.name, a]))
  const dims = locked ? specKeys(skus) : hasTemplate ? specAttrs.map((a) => a.name) : freeDims
  const parents = categories.filter((c) => !c.parentId).sort((a, b) => a.sort - b.sort)

  function setField<K extends keyof Base>(key: K, value: Base[K]) {
    setBase((prev) => ({ ...prev, [key]: value }))
  }

  function patchSku(key: number, patch: Partial<SkuDraft>) {
    setSkus((prev) => prev.map((sku) => (sku.key === key ? { ...sku, ...patch } : sku)))
  }

  function setSpec(key: number, dim: string, value: string) {
    setSkus((prev) => prev.map((sku) => (sku.key === key ? { ...sku, specs: { ...sku.specs, [dim]: value } } : sku)))
  }

  function paramValue(name: string) {
    return params.find((p) => p.name === name)?.value ?? ''
  }

  function setParamValue(name: string, value: string) {
    setParams((prev) => {
      if (prev.some((p) => p.name === name)) return prev.map((p) => (p.name === name ? { ...p, value } : p))
      return [...prev, { name, value }]
    })
  }

  function buildSkus(): { skus: SkuInput[]; priceChanges: { skuId: string; price: number }[] } | string {
    if (skus.length === 0) return '至少要有一个 SKU'
    const seen = new Set<string>()
    const out: SkuInput[] = []
    const priceChanges: { skuId: string; price: number }[] = []
    for (const [index, sku] of skus.entries()) {
      const label = `第 ${index + 1} 个 SKU`
      const price = yuanToCents(sku.price)
      if (price === null || price <= 0) return `${label} 售价要填大于 0 的元金额`
      const market = sku.marketPrice.trim() === '' ? 0 : yuanToCents(sku.marketPrice)
      if (market === null) return `${label} 划线价格式不对`
      if (locked) {
        out.push({
          id: sku.id,
          specs_json: sku.specsJson,
          price: sku.origPrice,
          market_price: yuanToCents(sku.marketPrice) ?? 0,
          image: sku.image,
          barcode: sku.barcode,
          enabled: sku.enabled,
        })
        if (price !== sku.origPrice) priceChanges.push({ skuId: sku.id, price })
        continue
      }
      const specs: Record<string, string> = {}
      for (const dim of dims) {
        const value = (sku.specs[dim] ?? '').trim()
        if (!value) return `${label} 的 ${dim} 没填`
        const attr = specAttrMap.get(dim)
        if (attr && !attr.allowCustom && !attr.options.includes(value)) return `${label} 的 ${dim} 只能从可选值里选`
        specs[dim] = value
      }
      const sig = dims.map((dim) => `${dim}=${specs[dim]}`).join('\n')
      if (seen.has(sig)) return `${label} 和前面的 SKU 规格重复`
      seen.add(sig)
      out.push({
        ...(sku.id ? { id: sku.id } : {}),
        specs_json: JSON.stringify(specs),
        price,
        market_price: market,
        image: sku.image.trim(),
        barcode: sku.barcode.trim(),
        enabled: sku.enabled,
      })
    }
    return { skus: out, priceChanges }
  }

  function buildInput(): { input: ProductInput; priceChanges: { skuId: string; price: number }[] } | string {
    if (!base.name.trim()) return '商品名称必填'
    if (!locked) {
      if (!base.categoryId) return '请选择分类'
      if (hasTemplate && attrsFor !== templateId) return '分类模板属性还没加载好 稍后再保存'
      if (category && !category.parentId) return '只能选二级分类'
      if (!base.brandId) return '请选择品牌'
    }
    const weight = base.weightGram.trim() === '' ? 0 : Number(base.weightGram)
    if (!Number.isInteger(weight) || weight < 0) return '重量要填非负整数克'
    const built = buildSkus()
    if (typeof built === 'string') return built
    const allowed = new Set(paramAttrs.map((a) => a.name))
    const cleanParams = params
      .map((p) => ({ name: p.name.trim(), value: p.value.trim() }))
      .filter((p) => p.name && p.value && (locked || !hasTemplate || allowed.has(p.name)))
    const cleanImages = images.map((url) => url.trim()).filter(Boolean)
    if (cleanImages.length > maxImages) return `图片最多 ${maxImages} 张`
    return {
      input: {
        category_id: base.categoryId,
        brand_id: base.brandId,
        name: base.name.trim(),
        subtitle: base.subtitle.trim(),
        keywords: base.keywords.trim(),
        unit: base.unit.trim(),
        weight_gram: weight,
        images: cleanImages,
        detail_html: detailHtml,
        skus: built.skus,
        params: cleanParams,
      },
      priceChanges: built.priceChanges,
    }
  }

  async function save(submitAfter: boolean) {
    clear()
    const built = buildInput()
    if (typeof built === 'string') {
      fail(null, built)
      return
    }
    setBusy(true)
    try {
      if (isNew) {
        const created = await createProduct(built.input)
        if (submitAfter && created.id) await submitProduct(created.id)
        ok(submitAfter ? '已创建并提交审核' : '已创建草稿')
        if (created.id) navigate(`/products/${encodeURIComponent(created.id)}`, { replace: true })
        return
      }
      await updateProduct(id, built.input)
      for (const change of built.priceChanges) await updateSkuPrice(id, change.skuId, change.price)
      if (submitAfter) await submitProduct(id)
      fill(await getProduct(id))
      const priced = built.priceChanges.length ? ` 改价 ${built.priceChanges.length} 个 SKU` : ''
      ok((submitAfter ? '已保存并提交审核' : '已保存') + priced)
    } catch (err) {
      fail(err, '保存失败')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <section><h1>商品编辑</h1><p>加载中</p><Notice {...note} /></section>

  const canSubmit = isNew || status === ProductStatus.Draft || status === ProductStatus.Rejected

  return (
    <section>
      <h1>{isNew ? '新建商品' : `编辑商品 ${base.name}`}</h1>
      <p className="hint">
        {isNew ? '保存后为草稿' : `状态 ${productStatusText[status] ?? status}`} <Link to="/products">返回列表</Link>
      </p>
      {locked ? <p className="warn">商品在售 只能改价格和文字 分类 品牌和规格结构已锁定 需要调整请先下架</p> : null}
      <div className="steps">
        {steps.map((name, index) => (
          <button key={name} type="button" className={index === step ? 'current' : ''} onClick={() => setStep(index)}>
            {index + 1} {name}
          </button>
        ))}
      </div>

      {step === 0 ? (
        <>
          <label>名称<input value={base.name} onChange={(e) => setField('name', e.target.value)} /></label>
          <label>副标题<input value={base.subtitle} onChange={(e) => setField('subtitle', e.target.value)} /></label>
          <label>关键词<input value={base.keywords} onChange={(e) => setField('keywords', e.target.value)} placeholder="空格分隔" /></label>
          <div className="row">
            <label>单位<input value={base.unit} onChange={(e) => setField('unit', e.target.value)} placeholder="件 盒 袋" /></label>
            <label>重量 克<input value={base.weightGram} onChange={(e) => setField('weightGram', e.target.value)} inputMode="numeric" /></label>
          </div>
          <div className="row">
            <label>
              分类
              <select value={base.categoryId} disabled={locked} onChange={(e) => setField('categoryId', e.target.value)}>
                <option value="">请选择二级分类</option>
                {parents.map((parent) => (
                  <optgroup key={parent.id} label={parent.name + (parent.visible ? '' : ' 隐藏')}>
                    {categories
                      .filter((c) => c.parentId === parent.id)
                      .sort((a, b) => a.sort - b.sort)
                      .map((c) => <option key={c.id} value={c.id}>{c.name}{c.visible ? '' : ' 隐藏'}</option>)}
                  </optgroup>
                ))}
              </select>
            </label>
            <label>
              品牌
              <select value={base.brandId} disabled={locked} onChange={(e) => setField('brandId', e.target.value)}>
                <option value="">请选择品牌</option>
                {brands.map((b) => <option key={b.id} value={b.id}>{b.name}{b.visible ? '' : ' 隐藏'}</option>)}
              </select>
            </label>
          </div>
          {base.categoryId ? <p className="hint">{hasTemplate ? '该分类绑定了属性模板 规格和参数按模板填写' : '该分类未绑定模板 规格和参数可自由填写'}</p> : null}
        </>
      ) : null}

      {step === 1 ? (
        <>
          <Notice text={attrsError} tone="error" />
          {!locked && hasTemplate ? (
            <p className="hint">{specAttrs.length ? `规格维度来自模板 ${specAttrs.map((a) => a.name).join(' ')}` : '模板没有规格属性 只能保留一个 SKU'}</p>
          ) : null}
          {!locked && !hasTemplate ? (
            <fieldset>
              <legend>规格维度</legend>
              <div className="actions">
                {freeDims.map((dim) => (
                  <span key={dim} className="tag">
                    {dim} <button type="button" onClick={() => setFreeDims((prev) => prev.filter((d) => d !== dim))}>移除</button>
                  </span>
                ))}
                <input value={newDim} onChange={(e) => setNewDim(e.target.value)} placeholder="如 颜色" />
                <button
                  type="button"
                  onClick={() => {
                    const name = newDim.trim()
                    if (name && !freeDims.includes(name)) setFreeDims((prev) => [...prev, name])
                    setNewDim('')
                  }}
                >
                  加维度
                </button>
              </div>
            </fieldset>
          ) : null}
          {specAttrs.map((a) => (a.options.length ? (
            <datalist key={a.id} id={`spec-${a.id}`}>{a.options.map((o) => <option key={o} value={o} />)}</datalist>
          ) : null))}
          <table>
            <thead>
              <tr>
                {dims.map((dim) => <th key={dim}>{dim}</th>)}
                <th>售价 元</th>
                <th>划线价 元</th>
                <th>图片地址</th>
                <th>条码</th>
                <th>启用</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {skus.map((sku) => (
                <tr key={sku.key}>
                  {dims.map((dim) => {
                    const attr = specAttrMap.get(dim)
                    const value = sku.specs[dim] ?? ''
                    if (!locked && attr && !attr.allowCustom && attr.options.length) {
                      return (
                        <td key={dim}>
                          <select value={value} onChange={(e) => setSpec(sku.key, dim, e.target.value)}>
                            <option value="">请选择</option>
                            {attr.options.map((o) => <option key={o} value={o}>{o}</option>)}
                          </select>
                        </td>
                      )
                    }
                    return (
                      <td key={dim}>
                        <input
                          value={value}
                          disabled={locked}
                          list={attr && attr.options.length ? `spec-${attr.id}` : undefined}
                          onChange={(e) => setSpec(sku.key, dim, e.target.value)}
                        />
                      </td>
                    )
                  })}
                  <td><input value={sku.price} onChange={(e) => patchSku(sku.key, { price: e.target.value })} inputMode="decimal" placeholder="0.00" /></td>
                  <td><input value={sku.marketPrice} disabled={locked} onChange={(e) => patchSku(sku.key, { marketPrice: e.target.value })} inputMode="decimal" /></td>
                  <td><input value={sku.image} disabled={locked} onChange={(e) => patchSku(sku.key, { image: e.target.value })} /></td>
                  <td><input value={sku.barcode} disabled={locked} onChange={(e) => patchSku(sku.key, { barcode: e.target.value })} /></td>
                  <td><input type="checkbox" checked={sku.enabled} disabled={locked} onChange={(e) => patchSku(sku.key, { enabled: e.target.checked })} /></td>
                  <td>
                    <button type="button" className="danger" disabled={locked || skus.length <= 1} onClick={() => setSkus((prev) => prev.filter((s) => s.key !== sku.key))}>
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <button type="button" disabled={locked} onClick={() => setSkus((prev) => [...prev, blankSku()])}>加一个 SKU</button>
          {locked ? <p className="hint">售价改动保存时逐个走改价接口 其他 SKU 字段在售期间不可改</p> : null}
        </>
      ) : null}

      {step === 2 ? (
        <>
          {hasTemplate && !locked ? (
            paramAttrs.length === 0 ? (
              <p className="hint">模板没有参数属性</p>
            ) : (
              paramAttrs.map((a) => (
                <label key={a.id}>
                  {a.name}
                  {!a.allowCustom && a.options.length ? (
                    <select value={paramValue(a.name)} onChange={(e) => setParamValue(a.name, e.target.value)}>
                      <option value="">不填</option>
                      {a.options.map((o) => <option key={o} value={o}>{o}</option>)}
                    </select>
                  ) : (
                    <>
                      <input value={paramValue(a.name)} list={a.options.length ? `param-${a.id}` : undefined} onChange={(e) => setParamValue(a.name, e.target.value)} />
                      {a.options.length ? <datalist id={`param-${a.id}`}>{a.options.map((o) => <option key={o} value={o} />)}</datalist> : null}
                    </>
                  )}
                </label>
              ))
            )
          ) : (
            <>
              <p className="hint">{locked ? '在售商品可修改参数文字' : '分类未绑定模板 参数名和值自由填写'}</p>
              {params.map((p, index) => (
                <div className="row" key={index}>
                  <label>参数名<input value={p.name} onChange={(e) => setParams((prev) => prev.map((q, i) => (i === index ? { ...q, name: e.target.value } : q)))} /></label>
                  <label>参数值<input value={p.value} onChange={(e) => setParams((prev) => prev.map((q, i) => (i === index ? { ...q, value: e.target.value } : q)))} /></label>
                  <button type="button" className="danger" onClick={() => setParams((prev) => prev.filter((_, i) => i !== index))}>删除</button>
                </div>
              ))}
              <button type="button" onClick={() => setParams((prev) => [...prev, { name: '', value: '' }])}>加一个参数</button>
            </>
          )}
        </>
      ) : null}

      {step === 3 ? (
        <>
          <h2>图片 {images.filter((u) => u.trim()).length} / {maxImages}</h2>
          <p className="hint">第一张作为主图</p>
          {images.map((url, index) => (
            <div className="row" key={index}>
              <label>图片 {index + 1}<input value={url} onChange={(e) => setImages((prev) => prev.map((u, i) => (i === index ? e.target.value : u)))} placeholder="https://" /></label>
              {url.trim() ? <img src={url.trim()} alt="" width={48} height={48} style={{ objectFit: 'cover' }} /> : null}
              <button type="button" className="danger" onClick={() => setImages((prev) => (prev.length > 1 ? prev.filter((_, i) => i !== index) : ['']))}>删除</button>
            </div>
          ))}
          <button type="button" disabled={images.length >= maxImages} onClick={() => setImages((prev) => [...prev, ''])}>加一张</button>
          <h2>详情 HTML</h2>
          <textarea value={detailHtml} onChange={(e) => setDetailHtml(e.target.value)} />
          {detailHtml.trim() ? <iframe className="preview" title="详情预览" sandbox="" srcDoc={detailHtml} /> : null}
        </>
      ) : null}

      <div className="actions">
        <button type="button" disabled={step === 0} onClick={() => setStep((n) => n - 1)}>上一步</button>
        <button type="button" disabled={step === steps.length - 1} onClick={() => setStep((n) => n + 1)}>下一步</button>
        <button type="button" disabled={busy} onClick={() => void save(false)}>保存</button>
        {canSubmit ? <button type="button" disabled={busy} onClick={() => void save(true)}>保存并提交审核</button> : null}
      </div>
      <Notice {...note} />
    </section>
  )
}
