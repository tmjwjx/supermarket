import { useState, type FormEvent } from 'react'
import {
  createRecommendation,
  deleteRecommendation,
  listAllProducts,
  listRecommendations,
  productStatusText,
  updateRecommendation,
  type ProductCard,
  type Recommendation,
  type RecommendationInput,
} from '../api/admin.ts'
import { formatTime, localInputToUnix, unixToLocalInput, useLoad, useNote } from './shared.ts'
import Notice from './Notice.tsx'

type Draft = { slot: string; productId: string; sort: string; start: string; end: string }

const slotText: Record<string, string> = { new: '新品', hot: '热卖', topic: '专题' }
const knownSlots = Object.keys(slotText)

function toDraft(r?: Recommendation): Draft {
  return {
    slot: r?.slot ?? 'new',
    productId: r?.productId ?? '',
    sort: String(r?.sort ?? 0),
    start: unixToLocalInput(r?.startAt ?? 0),
    end: unixToLocalInput(r?.endAt ?? 0),
  }
}

function toInput(d: Draft): RecommendationInput | string {
  if (!d.slot.trim()) return '位置必填'
  if (!d.productId) return '请选择商品'
  const start = localInputToUnix(d.start)
  const end = localInputToUnix(d.end)
  if (start && end && end <= start) return '结束时间要晚于开始时间'
  return { slot: d.slot.trim(), product_id: d.productId, sort: Number(d.sort) || 0, start_at_unix: start, end_at_unix: end }
}

function phase(r: Recommendation, now: number): { text: string; tone: string } {
  if (r.endAt && r.endAt <= now) return { text: '已过期', tone: 'off' }
  if (r.startAt && r.startAt > now) return { text: '未开始', tone: '' }
  return { text: '生效中', tone: 'on' }
}

function DraftFields({ value, onChange, products }: { value: Draft; onChange: (next: Draft) => void; products: ProductCard[] }) {
  return (
    <div className="row">
      <label>
        位置
        <input value={value.slot} list="slot-options" onChange={(e) => onChange({ ...value, slot: e.target.value })} />
      </label>
      <label>
        商品
        <select value={value.productId} onChange={(e) => onChange({ ...value, productId: e.target.value })}>
          <option value="">请选择</option>
          {products.map((p) => (
            <option key={p.id} value={p.id}>{p.name || p.id} {productStatusText[p.status] ?? ''}</option>
          ))}
        </select>
      </label>
      <label>排序<input value={value.sort} onChange={(e) => onChange({ ...value, sort: e.target.value })} inputMode="numeric" /></label>
      <label>开始 不填为立即<input type="datetime-local" value={value.start} onChange={(e) => onChange({ ...value, start: e.target.value })} /></label>
      <label>结束 不填为长期<input type="datetime-local" value={value.end} onChange={(e) => onChange({ ...value, end: e.target.value })} /></label>
    </div>
  )
}

export default function SlotsPage() {
  const recs = useLoad<Recommendation[]>(listRecommendations, [])
  const products = useLoad<ProductCard[]>(listAllProducts, [])
  const [draft, setDraft] = useState<Draft>(toDraft())
  const [editing, setEditing] = useState('')
  const [edit, setEdit] = useState<Draft>(toDraft())
  const [slotFilter, setSlotFilter] = useState('')
  const { note, ok, fail } = useNote()
  const [now] = useState(() => Math.floor(Date.now() / 1000))
  const names = new Map(products.data.map((p) => [p.id, p.name]))

  async function run(action: () => Promise<unknown>, done: string) {
    try {
      await action()
      ok(done)
      recs.reload()
      return true
    } catch (err) {
      fail(err, '操作失败')
      return false
    }
  }

  async function onCreate(event: FormEvent) {
    event.preventDefault()
    const input = toInput(draft)
    if (typeof input === 'string') return fail(null, input)
    if (await run(() => createRecommendation(input), '已加入推荐位')) setDraft({ ...toDraft(), slot: draft.slot })
  }

  async function onSave(id: string) {
    const input = toInput(edit)
    if (typeof input === 'string') return fail(null, input)
    if (await run(() => updateRecommendation(id, input), '推荐已修改')) setEditing('')
  }

  const slots = [...new Set([...knownSlots, ...recs.data.map((r) => r.slot)])]
  const list = recs.data
    .filter((r) => !slotFilter || r.slot === slotFilter)
    .sort((a, b) => a.slot.localeCompare(b.slot) || a.sort - b.sort)

  return (
    <section>
      <h1>推荐位</h1>
      <p className="hint">买家端只展示生效期内且在售的商品</p>
      <datalist id="slot-options">{slots.map((s) => <option key={s} value={s}>{slotText[s] ?? s}</option>)}</datalist>
      <Notice text={recs.error || products.error} tone="error" />
      <Notice {...note} />
      <form onSubmit={(event) => void onCreate(event)}>
        <DraftFields value={draft} onChange={setDraft} products={products.data} />
        <button type="submit">新建推荐</button>
      </form>
      <div className="row">
        <label>
          按位置筛选
          <select value={slotFilter} onChange={(e) => setSlotFilter(e.target.value)}>
            <option value="">全部</option>
            {slots.map((s) => <option key={s} value={s}>{slotText[s] ?? s}</option>)}
          </select>
        </label>
      </div>
      <table>
        <thead><tr><th>位置</th><th>商品</th><th>排序</th><th>开始</th><th>结束</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          {list.map((r) => {
            if (editing === r.id) {
              return (
                <tr key={r.id}>
                  <td colSpan={7}>
                    <DraftFields value={edit} onChange={setEdit} products={products.data} />
                    <div className="actions">
                      <button type="button" onClick={() => void onSave(r.id)}>保存</button>
                      <button type="button" onClick={() => setEditing('')}>取消</button>
                    </div>
                  </td>
                </tr>
              )
            }
            const p = phase(r, now)
            return (
              <tr key={r.id} className={p.tone === 'off' ? 'muted' : ''}>
                <td>{slotText[r.slot] ? `${slotText[r.slot]} ${r.slot}` : r.slot}</td>
                <td>{r.productName || names.get(r.productId) || r.productId}</td>
                <td>{r.sort}</td>
                <td>{r.startAt ? formatTime(r.startAt) : '立即'}</td>
                <td>{r.endAt ? formatTime(r.endAt) : '长期'}</td>
                <td><span className={`tag ${p.tone}`}>{p.text}</span></td>
                <td>
                  <div className="actions">
                    <button type="button" onClick={() => { setEditing(r.id); setEdit(toDraft(r)) }}>修改</button>
                    <button
                      type="button"
                      className="danger"
                      onClick={() => {
                        if (window.confirm('删除这条推荐')) void run(() => deleteRecommendation(r.id), '推荐已删除')
                      }}
                    >
                      删除
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      {!recs.loading && list.length === 0 ? <p>还没有推荐</p> : null}
    </section>
  )
}
